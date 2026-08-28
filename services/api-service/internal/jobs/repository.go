package jobs

import (
	"context"
	"errors"
	"sync"
)

var ErrJobNotFound = errors.New("job not found")

type MemoryRepository struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{jobs: make(map[string]Job)}
}

func (r *MemoryRepository) Save(_ context.Context, job Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = cloneJob(job)
	return nil
}

func (r *MemoryRepository) Find(_ context.Context, id string) (Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, ok := r.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return cloneJob(job), nil
}

func cloneJob(job Job) Job {
	job.InputFiles = append([]StoredFile(nil), job.InputFiles...)
	job.OutputFiles = append([]OutputFile(nil), job.OutputFiles...)
	return job
}
