// Package jobstore advances jobs and persists the reviewable report in
// PostgreSQL. api-service owns this schema; the worker only reads and writes
// rows of jobs it was handed and never creates or migrates tables.
package jobstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/services/document-service/internal/app/port"
	"order-fill/services/document-service/internal/domain/orderfill"
)

var (
	_ port.JobStore    = (*Store)(nil)
	_ port.ReportStore = (*Store)(nil)
)

// Store is the PostgreSQL backed job and report store.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) MarkProcessing(ctx context.Context, jobID string, at time.Time) error {
	const query = `UPDATE jobs
		SET status = 'processing', progress = 0.04, progress_message = 'Забираю файлы', updated_at = $2
		WHERE id = $1`
	tag, err := s.pool.Exec(ctx, query, jobID, at.UTC())
	if err != nil {
		return fmt.Errorf("mark job %s processing: %w", jobID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mark job %s processing: job not found", jobID)
	}
	return nil
}

func (s *Store) MarkFailed(ctx context.Context, jobID string, code string, message string, at time.Time) error {
	const query = `UPDATE jobs
		SET status = 'failed', error_code = $2, error_message = $3, progress = 1, progress_message = $3, updated_at = $4
		WHERE id = $1`
	tag, err := s.pool.Exec(ctx, query, jobID, code, message, at.UTC())
	if err != nil {
		return fmt.Errorf("mark job %s failed: %w", jobID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mark job %s failed: job not found", jobID)
	}
	return nil
}

func (s *Store) SaveResult(ctx context.Context, jobID string, status string, outputs []port.OutputFile, at time.Time) error {
	outputFiles, err := marshalJSON(outputFilesToDTO(outputs), "output_files")
	if err != nil {
		return err
	}

	const query = `UPDATE jobs
		SET status = $2, output_files = $3, updated_at = $4, error_code = NULL, error_message = NULL,
			progress = 1, progress_message = ''
		WHERE id = $1`
	tag, err := s.pool.Exec(ctx, query, jobID, status, outputFiles, at.UTC())
	if err != nil {
		return fmt.Errorf("save result for job %s: %w", jobID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("save result for job %s: job not found", jobID)
	}
	return nil
}

func (s *Store) Outputs(ctx context.Context, jobID string) ([]port.OutputFile, error) {
	const query = `SELECT output_files FROM jobs WHERE id = $1`

	var raw []byte
	if err := s.pool.QueryRow(ctx, query, jobID).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("select outputs for job %s: job not found", jobID)
		}
		return nil, fmt.Errorf("select outputs for job %s: %w", jobID, err)
	}

	var outputs []outputFileDTO
	if err := unmarshalJSON(raw, &outputs, "output_files"); err != nil {
		return nil, err
	}
	return outputFilesToDomain(outputs), nil
}

func (s *Store) SetProgress(ctx context.Context, jobID string, fraction float64, message string, at time.Time) error {
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	const query = `UPDATE jobs SET progress = $2, progress_message = $3, updated_at = $4 WHERE id = $1`
	tag, err := s.pool.Exec(ctx, query, jobID, fraction, message, at.UTC())
	if err != nil {
		return fmt.Errorf("set progress for job %s: %w", jobID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set progress for job %s: job not found", jobID)
	}
	return nil
}

func (s *Store) Save(ctx context.Context, jobID string, summary orderfill.Summary, rows []orderfill.ReportRow, at time.Time) error {
	summaryJSON, err := marshalJSON(summaryToDTO(summary), "summary")
	if err != nil {
		return err
	}
	rowsJSON, err := marshalJSON(reportRowsToDTO(rows), "rows")
	if err != nil {
		return err
	}

	const query = `INSERT INTO job_reports (job_id, summary, "rows", updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (job_id) DO UPDATE
		SET summary = EXCLUDED.summary, "rows" = EXCLUDED."rows", updated_at = EXCLUDED.updated_at`
	if _, err := s.pool.Exec(ctx, query, jobID, summaryJSON, rowsJSON, at.UTC()); err != nil {
		return fmt.Errorf("save report for job %s: %w", jobID, err)
	}
	return nil
}

func (s *Store) Load(ctx context.Context, jobID string) (orderfill.Summary, []orderfill.ReportRow, error) {
	const query = `SELECT summary, "rows" FROM job_reports WHERE job_id = $1`

	var summaryRaw, rowsRaw []byte
	if err := s.pool.QueryRow(ctx, query, jobID).Scan(&summaryRaw, &rowsRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return orderfill.Summary{}, nil, fmt.Errorf("select report for job %s: report not found", jobID)
		}
		return orderfill.Summary{}, nil, fmt.Errorf("select report for job %s: %w", jobID, err)
	}

	var summary summaryDTO
	if err := unmarshalJSON(summaryRaw, &summary, "summary"); err != nil {
		return orderfill.Summary{}, nil, err
	}
	var rows []reportRowDTO
	if err := unmarshalJSON(rowsRaw, &rows, "rows"); err != nil {
		return orderfill.Summary{}, nil, err
	}
	return summaryToDomain(summary), reportRowsToDomain(rows), nil
}
