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

func (s *Store) Record(e domain.Event) {
	_, _ = s.pool.Exec(context.Background(),
		`INSERT INTO audit_events (id, type, actor_id, company_id, job_id, created_at, payload)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.Type, e.ActorID, e.CompanyID, e.JobID, e.CreatedAt.UTC(), e.Payload)
}

func (s *Store) List(companyID string) []domain.Event {
	rows, err := s.pool.Query(context.Background(),
		`SELECT id, type, actor_id, company_id, job_id, created_at, payload FROM audit_events
		 WHERE ($1 = '' OR company_id = $1) ORDER BY created_at`, companyID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]domain.Event, 0)
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.ID, &e.Type, &e.ActorID, &e.CompanyID, &e.JobID, &e.CreatedAt, &e.Payload); err != nil {
			return out
		}
		e.CreatedAt = e.CreatedAt.UTC()
		out = append(out, e)
	}
	return out
}
