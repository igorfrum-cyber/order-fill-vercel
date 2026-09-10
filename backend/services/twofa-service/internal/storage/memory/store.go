package memory

import (
	"context"
	"sync"

	"order-fill/backend/services/twofa-service/internal/domain"
)

type Store struct {
	mu    sync.Mutex
	items map[string]domain.Credential
}

func NewStore() *Store {
	return &Store{items: map[string]domain.Credential{}}
}

func (s *Store) Save(_ context.Context, cred domain.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[cred.UserID] = cred
	return nil
}

func (s *Store) Get(_ context.Context, userID string) (domain.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cred, ok := s.items[userID]
	if !ok {
		return domain.Credential{}, domain.ErrNotFound
	}
	return cred, nil
}

func (s *Store) Delete(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, userID)
	return nil
}
