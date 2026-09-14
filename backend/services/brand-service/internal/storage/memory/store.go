package memory

import (
	"context"
	"sync"

	"order-fill/backend/services/brand-service/internal/domain"
)

type Store struct {
	mu       sync.RWMutex
	policies map[string]domain.Policy
}

func New() *Store { return &Store{policies: make(map[string]domain.Policy)} }

func (s *Store) Get(_ context.Context, brand string) (domain.Policy, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.policies[brand]
	return policy, ok, nil
}

func (s *Store) Save(_ context.Context, policy domain.Policy, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[policy.Key] = policy
	return nil
}
