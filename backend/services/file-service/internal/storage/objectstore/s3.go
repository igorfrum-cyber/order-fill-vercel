package objectstore

import (
	"bytes"
	"context"
	"io"
	"sync"

	"order-fill/backend/services/file-service/internal/domain"
)

type blob struct {
	body        []byte
	contentType string
}

// Store is an in-memory object blob store.
// ponytail: swap for MinIO/S3 PutObject/GetObject when compose wires the bucket.
type Store struct {
	mu    sync.Mutex
	items map[string]blob
}

func NewS3() *Store {
	return &Store{items: map[string]blob{}}
}

func (s *Store) Put(_ context.Context, key string, body io.Reader, _ int64, contentType string) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = blob{body: bytes.Clone(raw), contentType: contentType}
	return nil
}

func (s *Store) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok {
		return nil, "", domain.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(item.body)), item.contentType, nil
}
