package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/audit-service/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Record(ctx context.Context, e domain.Event) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_events (id, type, actor_id, company_id, job_id, created_at, payload)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.Type, e.ActorID, e.CompanyID, e.JobID, e.CreatedAt.UTC(), e.Payload)
	return err
}

func (s *Store) List(ctx context.Context, companyID string) ([]domain.Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, type, actor_id, company_id, job_id, created_at, payload FROM audit_events
		 WHERE ($1 = '' OR company_id = $1) ORDER BY created_at`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Event, 0)
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.ID, &e.Type, &e.ActorID, &e.CompanyID, &e.JobID, &e.CreatedAt, &e.Payload); err != nil {
			return out, err
		}
		e.CreatedAt = e.CreatedAt.UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}
