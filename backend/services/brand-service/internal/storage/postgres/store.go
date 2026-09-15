package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/brand-service/internal/domain"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Get(ctx context.Context, brand string) (domain.Policy, bool, error) {
	var body []byte
	err := s.pool.QueryRow(ctx, `SELECT policy FROM brand_policy_overrides WHERE brand = $1`, brand).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Policy{}, false, nil
	}
	if err != nil {
		return domain.Policy{}, false, fmt.Errorf("get brand policy: %w", err)
	}
	var policy domain.Policy
	if err := json.Unmarshal(body, &policy); err != nil {
		return domain.Policy{}, false, fmt.Errorf("decode brand policy: %w", err)
	}
	return policy, true, nil
}

func (s *Store) Save(ctx context.Context, policy domain.Policy, actorID string) error {
	body, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("encode brand policy: %w", err)
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO brand_policy_overrides (brand, policy, updated_by, updated_at)
		VALUES ($1, $2, $3, now()) ON CONFLICT (brand) DO UPDATE
		SET policy = EXCLUDED.policy, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at`,
		policy.Key, body, actorID)
	if err != nil {
		return fmt.Errorf("save brand policy: %w", err)
	}
	return nil
}
