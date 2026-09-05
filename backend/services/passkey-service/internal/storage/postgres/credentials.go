package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/passkey-service/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Save(ctx context.Context, cred domain.PasskeyCredential) error {
	if err := domain.AssertPasskeyCredentialJSON(cred.Raw); err != nil {
		return err
	}
	transports, err := json.Marshal(cred.Transports)
	if err != nil {
		return err
	}
	if cred.Transports == nil {
		transports = []byte("[]")
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO passkey_credentials (id, user_id, name, public_key, sign_count, transports, aaguid, raw, created_at, last_used_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		cred.ID, cred.UserID, cred.Name, cred.PublicKey, int64(cred.SignCount), transports, cred.AAGUID, cred.Raw, cred.CreatedAt.UTC(), cred.LastUsedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (domain.PasskeyCredential, error) {
	cred, err := scanCred(s.pool.QueryRow(ctx, credSelect+` WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PasskeyCredential{}, domain.ErrNotFound
	}
	return cred, err
}

func (s *Store) List(ctx context.Context, userID string) ([]domain.PasskeyCredential, error) {
	rows, err := s.pool.Query(ctx, credSelect+` WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PasskeyCredential, 0)
	for rows.Next() {
		cred, err := scanCred(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cred)
	}
	return out, rows.Err()
}

func (s *Store) Update(ctx context.Context, cred domain.PasskeyCredential) error {
	transports, err := json.Marshal(cred.Transports)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE passkey_credentials SET name=$2, public_key=$3, sign_count=$4, transports=$5, aaguid=$6, raw=$7, last_used_at=$8 WHERE id=$1`,
		cred.ID, cred.Name, cred.PublicKey, int64(cred.SignCount), transports, cred.AAGUID, cred.Raw, cred.LastUsedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, userID, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM passkey_credentials WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const credSelect = `SELECT id, user_id, name, public_key, sign_count, transports, aaguid, raw, created_at, last_used_at FROM passkey_credentials`

type scanner interface {
	Scan(dest ...any) error
}

func scanCred(row scanner) (domain.PasskeyCredential, error) {
	var cred domain.PasskeyCredential
	var signCount int64
	var transports []byte
	if err := row.Scan(&cred.ID, &cred.UserID, &cred.Name, &cred.PublicKey, &signCount, &transports, &cred.AAGUID, &cred.Raw, &cred.CreatedAt, &cred.LastUsedAt); err != nil {
		return domain.PasskeyCredential{}, err
	}
	cred.SignCount = uint32(signCount)
	cred.CreatedAt = cred.CreatedAt.UTC()
	if len(transports) > 0 {
		_ = json.Unmarshal(transports, &cred.Transports)
	}
	if cred.Transports == nil {
		cred.Transports = []string{}
	}
	return cred, nil
}
