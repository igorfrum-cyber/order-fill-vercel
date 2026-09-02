package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
)

var (
	_ port.JobRepository = (*Repository)(nil)
	_ port.ReportReader  = (*Repository)(nil)
	_ port.IdentityStore = (*Repository)(nil)
)

const jobInsertColumns = `id, type, status, brand, order_month, created_at, updated_at,
		error_code, error_message, input_files, output_files, progress, progress_message, company_id, created_by`

const jobSelectColumns = `id, type, status, brand, order_month, created_at, updated_at,
		error_code, error_message, input_files, output_files, progress, progress_message,
		COALESCE(company_id, ''), COALESCE(created_by, '')`

// Repository is the PostgreSQL backed job store. api-service owns this schema;
// document-service only updates existing rows.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Migrate creates the schema. It is idempotent and safe to run on every start.
func migrateStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			brand TEXT NOT NULL DEFAULT '',
			order_month TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			error_code TEXT,
			error_message TEXT,
			input_files JSONB NOT NULL DEFAULT '[]',
			output_files JSONB NOT NULL DEFAULT '[]',
			progress DOUBLE PRECISION NOT NULL DEFAULT 0,
			progress_message TEXT NOT NULL DEFAULT ''
		)`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS progress DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS progress_message TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS companies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			disabled_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			company_id TEXT REFERENCES companies(id),
			login TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			disabled_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token_hash TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS invite_tokens (
			token_hash TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id TEXT PRIMARY KEY,
			at TIMESTAMPTZ NOT NULL,
			actor_id TEXT NOT NULL,
			action TEXT NOT NULL,
			company_id TEXT,
			job_id TEXT
		)`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS company_id TEXT`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS created_by TEXT`,
		`ALTER TABLE companies ADD COLUMN IF NOT EXISTS login_slug TEXT`,
		`UPDATE companies SET login_slug = 'company-' || lower(left(replace(id, '-', ''), 8))
			WHERE login_slug IS NULL OR login_slug = ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS companies_login_slug_uidx ON companies(login_slug) WHERE login_slug IS NOT NULL`,
		`ALTER TABLE companies ADD COLUMN IF NOT EXISTS logo_content_type TEXT`,
		`CREATE TABLE IF NOT EXISTS job_reports (
			job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
			summary JSONB NOT NULL DEFAULT '{}',
			"rows" JSONB NOT NULL DEFAULT '[]',
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_totp (
			user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			secret TEXT NOT NULL,
			enabled_at TIMESTAMPTZ,
			recovery_code_hashes JSONB NOT NULL DEFAULT '[]'
		)`,
		`CREATE TABLE IF NOT EXISTS login_challenges (
			token_hash TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL
		)`,
	}
}

func (r *Repository) Migrate(ctx context.Context) error {
	statements := migrateStatements()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply migration: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}

func (r *Repository) Create(ctx context.Context, entity job.Job) error {
	inputFiles, err := marshalJSON(inputFilesToDTO(entity.InputFiles), "input_files")
	if err != nil {
		return err
	}
	outputFiles, err := marshalJSON(outputFilesToDTO(entity.OutputFiles), "output_files")
	if err != nil {
		return err
	}

	var errorCode, errorMessage *string
	if entity.Failure != nil {
		errorCode = &entity.Failure.Code
		errorMessage = &entity.Failure.Message
	}

	const query = `INSERT INTO jobs (` + jobInsertColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err = r.pool.Exec(ctx, query,
		entity.ID,
		string(entity.Type),
		string(entity.Status),
		entity.Brand,
		entity.OrderMonth,
		entity.CreatedAt,
		entity.UpdatedAt,
		errorCode,
		errorMessage,
		inputFiles,
		outputFiles,
		entity.Progress,
		entity.ProgressMessage,
		nullIfEmpty(entity.CompanyID),
		nullIfEmpty(entity.CreatedBy),
	)
	if err != nil {
		return fmt.Errorf("insert job %s: %w", entity.ID, err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id string) (job.Job, error) {
	const query = `SELECT ` + jobSelectColumns + ` FROM jobs WHERE id = $1`
	entity, err := scanJob(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.Job{}, fmt.Errorf("%w: job %s", job.ErrNotFound, id)
		}
		return job.Job{}, fmt.Errorf("select job %s: %w", id, err)
	}
	return entity, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status job.Status, updatedAt time.Time) (job.Job, error) {
	const query = `UPDATE jobs SET status = $2, updated_at = $3 WHERE id = $1
		RETURNING ` + jobSelectColumns
	entity, err := scanJob(r.pool.QueryRow(ctx, query, id, string(status), updatedAt.UTC()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.Job{}, fmt.Errorf("%w: job %s", job.ErrNotFound, id)
		}
		return job.Job{}, fmt.Errorf("update job %s status: %w", id, err)
	}
	return entity, nil
}

func (r *Repository) Report(ctx context.Context, jobID string) (job.Report, error) {
	const query = `SELECT summary, "rows" FROM job_reports WHERE job_id = $1`

	var summaryRaw, rowsRaw []byte
	if err := r.pool.QueryRow(ctx, query, jobID).Scan(&summaryRaw, &rowsRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.Report{}, fmt.Errorf("%w: report for job %s", job.ErrNotFound, jobID)
		}
		return job.Report{}, fmt.Errorf("select report for job %s: %w", jobID, err)
	}

	var summary summaryDTO
	if err := unmarshalJSON(summaryRaw, &summary, "summary"); err != nil {
		return job.Report{}, err
	}
	var rows []reportRowDTO
	if err := unmarshalJSON(rowsRaw, &rows, "rows"); err != nil {
		return job.Report{}, err
	}

	return job.Report{
		JobID:   jobID,
		Summary: summaryToDomain(summary),
		Rows:    reportRowsToDomain(rows),
	}, nil
}

func scanJob(row pgx.Row) (job.Job, error) {
	var (
		entity                  job.Job
		jobType, status         string
		errorCode, errorMessage *string
		inputFiles, outputFiles []byte
	)
	err := row.Scan(
		&entity.ID,
		&jobType,
		&status,
		&entity.Brand,
		&entity.OrderMonth,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&errorCode,
		&errorMessage,
		&inputFiles,
		&outputFiles,
		&entity.Progress,
		&entity.ProgressMessage,
		&entity.CompanyID,
		&entity.CreatedBy,
	)
	if err != nil {
		return job.Job{}, err
	}

	return finishJobScan(entity, jobType, status, errorCode, errorMessage, inputFiles, outputFiles)
}

func scanListedJob(row pgx.Row) (job.Job, string, error) {
	var (
		entity                  job.Job
		jobType, status         string
		errorCode, errorMessage *string
		inputFiles, outputFiles []byte
		login                   string
	)
	err := row.Scan(
		&entity.ID,
		&jobType,
		&status,
		&entity.Brand,
		&entity.OrderMonth,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&errorCode,
		&errorMessage,
		&inputFiles,
		&outputFiles,
		&entity.Progress,
		&entity.ProgressMessage,
		&entity.CompanyID,
		&entity.CreatedBy,
		&login,
	)
	if err != nil {
		return job.Job{}, "", err
	}
	entity, err = finishJobScan(entity, jobType, status, errorCode, errorMessage, inputFiles, outputFiles)
	if err != nil {
		return job.Job{}, "", err
	}
	return entity, login, nil
}

func finishJobScan(entity job.Job, jobType string, status string, errorCode *string, errorMessage *string, inputFiles []byte, outputFiles []byte) (job.Job, error) {
	entity.Type = job.Type(jobType)
	entity.Status = job.Status(status)
	entity.CreatedAt = entity.CreatedAt.UTC()
	entity.UpdatedAt = entity.UpdatedAt.UTC()
	if errorCode != nil || errorMessage != nil {
		entity.Failure = &job.Failure{
			Code:    derefString(errorCode),
			Message: derefString(errorMessage),
		}
	}
	var inputDTOs []inputFileDTO
	if err := unmarshalJSON(inputFiles, &inputDTOs, "input_files"); err != nil {
		return job.Job{}, err
	}
	var outputDTOs []outputFileDTO
	if err := unmarshalJSON(outputFiles, &outputDTOs, "output_files"); err != nil {
		return job.Job{}, err
	}
	entity.InputFiles = inputFilesToDomain(inputDTOs)
	entity.OutputFiles = outputFilesToDomain(outputDTOs)
	return entity, nil
}

func (r *Repository) List(ctx context.Context, filter port.JobListFilter) ([]port.JobListRow, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	const query = `SELECT j.id, j.type, j.status, j.brand, j.order_month, j.created_at, j.updated_at,
		j.error_code, j.error_message, j.input_files, j.output_files, j.progress, j.progress_message,
		COALESCE(j.company_id, ''), COALESCE(j.created_by, ''), COALESCE(u.login, '')
		FROM jobs j
		LEFT JOIN users u ON u.id = j.created_by
		WHERE ($1 = '' OR j.company_id = $1)
		  AND ($2 = '' OR j.created_by = $2)
		ORDER BY j.created_at DESC
		LIMIT $3`
	rows, err := r.pool.Query(ctx, query, filter.CompanyID, filter.CreatedBy, limit)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()
	items := make([]port.JobListRow, 0)
	for rows.Next() {
		entity, login, err := scanListedJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, port.JobListRow{Job: entity, CreatedByLogin: login})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return items, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
