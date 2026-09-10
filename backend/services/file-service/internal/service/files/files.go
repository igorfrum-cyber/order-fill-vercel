package files

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"

	"order-fill/backend/services/file-service/internal/domain"
)

type BlobStore interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
}

type MetaStore interface {
	SaveObject(ctx context.Context, obj domain.Object) error
	GetByID(ctx context.Context, id string) (domain.Object, error)
	GetByKey(ctx context.Context, key string) (domain.Object, error)
	SaveUpload(ctx context.Context, up domain.Upload) error
	GetUpload(ctx context.Context, id string) (domain.Upload, error)
}

type Service struct {
	blobs BlobStore
	meta  MetaStore
}

func New(blobs BlobStore, meta MetaStore) *Service {
	return &Service{blobs: blobs, meta: meta}
}

func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
