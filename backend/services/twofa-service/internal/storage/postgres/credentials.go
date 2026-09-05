package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/twofa-service/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Save(ctx context.Context, cred domain.Credential) error {
	hashes, err := json.Marshal(cred.RecoveryCodeHashes)
	if err != nil {
		return err
	}
	if cred.RecoveryCodeHashes == nil {
		hashes = []byte("[]")
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO user_totp (user_id, secret, enabled, recovery_code_hashes)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) DO UPDATE SET secret = EXCLUDED.secret, enabled = EXCLUDED.enabled, recovery_code_hashes = EXCLUDED.recovery_code_hashes`,
		cred.UserID, cred.Secret, cred.Enabled, hashes)
	return err
}

func (s *Store) Get(ctx context.Context, userID string) (domain.Credential, error) {
	var cred domain.Credential
	var hashes []byte
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, secret, enabled, COALESCE(recovery_code_hashes, '[]'::jsonb) FROM user_totp WHERE user_id = $1`,
		userID).Scan(&cred.UserID, &cred.Secret, &cred.Enabled, &hashes)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Credential{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Credential{}, err
	}
	if len(hashes) > 0 {
		_ = json.Unmarshal(hashes, &cred.RecoveryCodeHashes)
	}
	if cred.RecoveryCodeHashes == nil {
		cred.RecoveryCodeHashes = []string{}
	}
	return cred, nil
}

func (s *Store) Delete(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM user_totp WHERE user_id = $1`, userID)
	return err
}
