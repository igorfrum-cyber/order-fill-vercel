package audit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"order-fill/backend/services/audit-service/internal/domain"
)

type Store interface {
	Record(ctx context.Context, e domain.Event) error
	List(ctx context.Context, companyID string) ([]domain.Event, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

func (s *Service) Record(ctx context.Context, typ, actorID, companyID, jobID, payload string) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	if err := s.store.Record(ctx, domain.Event{
		ID: id, Type: typ, ActorID: actorID, CompanyID: companyID, JobID: jobID,
		CreatedAt: s.now().UTC(), Payload: payload,
	}); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) List(ctx context.Context, companyID string) ([]domain.Event, error) {
	return s.store.List(ctx, companyID)
}

func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
