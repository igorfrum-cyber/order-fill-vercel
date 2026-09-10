package memory

import (
	"context"
	"sync"

	"order-fill/backend/services/passkey-service/internal/domain"
)

type Store struct {
	mu    sync.Mutex
	items map[string]domain.PasskeyCredential
}

func NewStore() *Store {
	return &Store{items: map[string]domain.PasskeyCredential{}}
}

func (s *Store) Save(_ context.Context, cred domain.PasskeyCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := domain.AssertPasskeyCredentialJSON(cred.Raw); err != nil {
		return err
	}
	if _, ok := s.items[cred.ID]; ok {
		return domain.ErrConflict
	}
	s.items[cred.ID] = cred
	return nil
}

func (s *Store) Get(_ context.Context, id string) (domain.PasskeyCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cred, ok := s.items[id]
	if !ok {
		return domain.PasskeyCredential{}, domain.ErrNotFound
	}
	return cred, nil
}

func (s *Store) List(_ context.Context, userID string) ([]domain.PasskeyCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.PasskeyCredential, 0)
	for _, cred := range s.items {
		if cred.UserID == userID {
			out = append(out, cred)
		}
	}
	return out, nil
}

func (s *Store) Update(_ context.Context, cred domain.PasskeyCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[cred.ID]; !ok {
		return domain.ErrNotFound
	}
	s.items[cred.ID] = cred
	return nil
}

func (s *Store) Delete(_ context.Context, userID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cred, ok := s.items[id]
	if !ok || cred.UserID != userID {
		return domain.ErrNotFound
	}
	delete(s.items, id)
	return nil
}
