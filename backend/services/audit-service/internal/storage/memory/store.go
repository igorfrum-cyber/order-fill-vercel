package memory

import (
	"context"
	"sync"

	"order-fill/backend/services/audit-service/internal/domain"
)

type Store struct {
	mu     sync.Mutex
	events []domain.Event
}

func New() *Store { return &Store{} }

func (s *Store) Record(_ context.Context, e domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *Store) List(_ context.Context, companyID string) ([]domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Event, 0)
	for _, e := range s.events {
		if companyID == "" || e.CompanyID == companyID {
			out = append(out, e)
		}
	}
	return out, nil
}
