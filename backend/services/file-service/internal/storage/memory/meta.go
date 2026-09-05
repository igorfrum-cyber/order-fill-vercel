package memory

import (
	"sync"

	"order-fill/backend/services/file-service/internal/domain"
)

type Meta struct {
	mu      sync.Mutex
	objects map[string]domain.Object
	keys    map[string]string
	uploads map[string]domain.Upload
}

func NewMeta() *Meta {
	return &Meta{objects: map[string]domain.Object{}, keys: map[string]string{}, uploads: map[string]domain.Upload{}}
}

func (m *Meta) SaveObject(obj domain.Object) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[obj.ID] = obj
	m.keys[obj.Key] = obj.ID
	return nil
}

func (m *Meta) GetByID(id string) (domain.Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[id]
	if !ok {
		return domain.Object{}, domain.ErrNotFound
	}
	return obj, nil
}

func (m *Meta) GetByKey(key string) (domain.Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.keys[key]
	if !ok {
		return domain.Object{}, domain.ErrNotFound
	}
	return m.objects[id], nil
}

func (m *Meta) SaveUpload(up domain.Upload) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploads[up.ID] = up
	return nil
}

func (m *Meta) GetUpload(id string) (domain.Upload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	up, ok := m.uploads[id]
	if !ok {
		return domain.Upload{}, domain.ErrNotFound
	}
	return up, nil
}
