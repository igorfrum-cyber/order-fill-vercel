package jobs

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"order-fill/backend/services/job-service/internal/domain"
	"order-fill/backend/services/job-service/internal/queue"
)

type Store interface {
	Create(ctx context.Context, job domain.Job) error
	Get(ctx context.Context, id string) (domain.Job, error)
	List(ctx context.Context) ([]domain.Job, error)
	Update(ctx context.Context, job domain.Job) error
	SaveReport(ctx context.Context, jobID string, report domain.Report) error
	GetReport(ctx context.Context, jobID string) (domain.Report, error)
}

type Files interface {
	Describe(ctx context.Context, ids []string) ([]domain.FileRef, error)
}

type Companies interface {
	MatchingMode(ctx context.Context, actor domain.Actor) (domain.MatchingMode, error)
}

type Publisher interface {
	Publish(msg queue.Message) error
}

type Service struct {
	store     Store
	files     Files
	companies Companies
	publisher Publisher
	now       func() time.Time
}

func New(store Store, files Files, companies Companies, publisher Publisher, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, files: files, companies: companies, publisher: publisher, now: now}
}

func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
