package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/job-service/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, job domain.Job) error {
	in, err := json.Marshal(job.InputFiles)
	if err != nil {
		return err
	}
	out, err := json.Marshal(job.OutputFiles)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO jobs (id, type, status, owner_user_id, company_id, matching_mode, created_at, updated_at, error_message, progress, progress_message, input_files, output_files)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		job.ID, string(job.Type), string(job.Status), job.OwnerUserID, job.CompanyID, string(job.MatchingMode),
		job.CreatedAt.UTC(), job.UpdatedAt.UTC(), job.ErrorMessage, job.Progress, job.ProgressMessage, in, out)
	if err != nil {
		return mapConflict(err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (domain.Job, error) {
	job, err := scanJob(s.pool.QueryRow(ctx, jobSelect+` WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, domain.ErrNotFound
	}
	return job, err
}

func (s *Store) List(ctx context.Context) ([]domain.Job, error) {
	rows, err := s.pool.Query(ctx, jobSelect+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (s *Store) Update(ctx context.Context, job domain.Job) error {
	in, err := json.Marshal(job.InputFiles)
	if err != nil {
		return err
	}
	out, err := json.Marshal(job.OutputFiles)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE jobs SET type=$2, status=$3, owner_user_id=$4, company_id=$5, matching_mode=$6, created_at=$7, updated_at=$8,
		 error_message=$9, progress=$10, progress_message=$11, input_files=$12, output_files=$13 WHERE id=$1`,
		job.ID, string(job.Type), string(job.Status), job.OwnerUserID, job.CompanyID, string(job.MatchingMode),
		job.CreatedAt.UTC(), job.UpdatedAt.UTC(), job.ErrorMessage, job.Progress, job.ProgressMessage, in, out)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) SaveReport(ctx context.Context, jobID string, report domain.Report) error {
	payload, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO job_reports (job_id, payload) VALUES ($1, $2)
		 ON CONFLICT (job_id) DO UPDATE SET payload = EXCLUDED.payload`, jobID, payload)
	return err
}

func (s *Store) GetReport(ctx context.Context, jobID string) (domain.Report, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM job_reports WHERE job_id = $1`, jobID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Report{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Report{}, err
	}
	var report domain.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return domain.Report{}, err
	}
	return report, nil
}

const jobSelect = `SELECT id, type, status, owner_user_id, company_id, matching_mode, created_at, updated_at,
	error_message, progress, progress_message, input_files, output_files FROM jobs`

type scanner interface {
	Scan(dest ...any) error
}

func scanJob(row scanner) (domain.Job, error) {
	var job domain.Job
	var typ, status, mode string
	var inRaw, outRaw []byte
	if err := row.Scan(&job.ID, &typ, &status, &job.OwnerUserID, &job.CompanyID, &mode,
		&job.CreatedAt, &job.UpdatedAt, &job.ErrorMessage, &job.Progress, &job.ProgressMessage, &inRaw, &outRaw); err != nil {
		return domain.Job{}, err
	}
	job.Type = domain.Type(typ)
	job.Status = domain.Status(status)
	job.MatchingMode = domain.ParseMatchingMode(mode)
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	if len(inRaw) > 0 {
		_ = json.Unmarshal(inRaw, &job.InputFiles)
	}
	if len(outRaw) > 0 {
		_ = json.Unmarshal(outRaw, &job.OutputFiles)
	}
	if job.InputFiles == nil {
		job.InputFiles = []domain.FileRef{}
	}
	if job.OutputFiles == nil {
		job.OutputFiles = []domain.FileRef{}
	}
	return job, nil
}

func mapConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}
