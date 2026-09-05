package audit

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"order-fill/backend/services/audit-service/internal/domain"
)

type Store interface {
	Record(e domain.Event)
	List(companyID string) []domain.Event
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

func (s *Service) Record(typ, actorID, companyID, jobID, payload string) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	s.store.Record(domain.Event{
		ID: id, Type: typ, ActorID: actorID, CompanyID: companyID, JobID: jobID,
		CreatedAt: s.now().UTC(), Payload: payload,
	})
	return id, nil
}

func (s *Service) List(companyID string) []domain.Event {
	return s.store.List(companyID)
}

func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
