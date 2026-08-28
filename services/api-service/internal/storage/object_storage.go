package storage

import (
	"context"
	"fmt"
	"io"
	"sync"

	"order-fill/services/api-service/internal/jobs"
)

type MemoryObjectStorage struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMemoryObjectStorage() *MemoryObjectStorage {
	return &MemoryObjectStorage{objects: make(map[string][]byte)}
}

func (s *MemoryObjectStorage) Put(_ context.Context, key string, file jobs.UploadFile) (jobs.StoredFile, error) {
	content, err := io.ReadAll(file.Reader)
	if err != nil {
		return jobs.StoredFile{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = append([]byte(nil), content...)

	return jobs.StoredFile{
		ID:          fmt.Sprintf("object:%s", key),
		Name:        file.Name,
		ContentType: file.ContentType,
		SizeBytes:   int64(len(content)),
		Key:         key,
	}, nil
}

func (s *MemoryObjectStorage) Get(_ context.Context, key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, ok := s.objects[key]
	return append([]byte(nil), content...), ok
}
