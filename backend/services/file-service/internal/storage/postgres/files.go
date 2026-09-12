package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/file-service/internal/domain"
)

type Meta struct {
	pool *pgxpool.Pool
}

func NewMeta(pool *pgxpool.Pool) *Meta {
	return &Meta{pool: pool}
}

func (m *Meta) SaveObject(ctx context.Context, obj domain.Object) error {
	_, err := m.pool.Exec(ctx,
		`INSERT INTO objects (id, key, name, content_type, size, company_id) VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (key) DO UPDATE SET id = EXCLUDED.id, name = EXCLUDED.name, content_type = EXCLUDED.content_type, size = EXCLUDED.size, company_id = EXCLUDED.company_id`,
		obj.ID, obj.Key, obj.Name, obj.ContentType, obj.Size, obj.CompanyID)
	if err != nil {
		return fmt.Errorf("save object: %w", err)
	}
	return nil
}

func (m *Meta) GetByID(ctx context.Context, id string) (domain.Object, error) {
	return m.scan(m.pool.QueryRow(ctx,
		`SELECT id, key, name, content_type, size, company_id FROM objects WHERE id = $1`, id))
}

func (m *Meta) GetByKey(ctx context.Context, key string) (domain.Object, error) {
	return m.scan(m.pool.QueryRow(ctx,
		`SELECT id, key, name, content_type, size, company_id FROM objects WHERE key = $1`, key))
}

func (m *Meta) SaveUpload(ctx context.Context, up domain.Upload) error {
	_, err := m.pool.Exec(ctx,
		`INSERT INTO uploads (id, name, content_type, object_id) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, content_type = EXCLUDED.content_type, object_id = EXCLUDED.object_id`,
		up.ID, up.Name, up.ContentType, nullIfEmpty(up.ObjectID))
	if err != nil {
		return fmt.Errorf("save upload: %w", err)
	}
	return nil
}

func (m *Meta) GetUpload(ctx context.Context, id string) (domain.Upload, error) {
	var up domain.Upload
	var objectID *string
	err := m.pool.QueryRow(ctx,
		`SELECT id, name, content_type, object_id FROM uploads WHERE id = $1`, id).Scan(&up.ID, &up.Name, &up.ContentType, &objectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Upload{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Upload{}, err
	}
	if objectID != nil {
		up.ObjectID = *objectID
	}
	return up, nil
}

func (m *Meta) scan(row pgx.Row) (domain.Object, error) {
	var obj domain.Object
	err := row.Scan(&obj.ID, &obj.Key, &obj.Name, &obj.ContentType, &obj.Size, &obj.CompanyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Object{}, domain.ErrNotFound
	}
	return obj, err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
