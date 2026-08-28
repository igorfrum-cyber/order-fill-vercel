package reports

import (
	"context"

	"order-fill/services/document-service/internal/orderfill"
)

type Repository interface {
	Save(ctx context.Context, jobID string, report orderfill.Report) error
}

type MemoryRepository struct {
	Reports map[string]orderfill.Report
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{Reports: map[string]orderfill.Report{}}
}

func (r *MemoryRepository) Save(_ context.Context, jobID string, report orderfill.Report) error {
	r.Reports[jobID] = report
	return nil
}
