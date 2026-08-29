package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
)

var (
	_ port.JobRepository = (*Repository)(nil)
	_ port.ReportReader  = (*Repository)(nil)
)

const jobColumns = `id, type, status, brand, order_month, created_at, updated_at,
		error_code, error_message, input_files, output_files`

// Repository is the PostgreSQL backed job store. api-service owns this schema;
// document-service only updates existing rows.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Migrate creates the schema. It is idempotent and safe to run on every start.
func (r *Repository) Migrate(ctx context.Context) error {
	statements := []string{
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
			output_files JSONB NOT NULL DEFAULT '[]'
		)`,
		`CREATE TABLE IF NOT EXISTS job_reports (
			job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
			summary JSONB NOT NULL DEFAULT '{}',
			"rows" JSONB NOT NULL DEFAULT '[]',
			updated_at TIMESTAMPTZ NOT NULL
		)`,
	}

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

	const query = `INSERT INTO jobs (` + jobColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
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
	)
	if err != nil {
		return fmt.Errorf("insert job %s: %w", entity.ID, err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id string) (job.Job, error) {
	const query = `SELECT ` + jobColumns + ` FROM jobs WHERE id = $1`
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
		RETURNING ` + jobColumns
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
	)
	if err != nil {
		return job.Job{}, err
	}

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

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
