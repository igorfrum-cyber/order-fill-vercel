package memory

import (
	"context"
	"sync"

	"order-fill/backend/services/job-service/internal/domain"
)

type Store struct {
	mu      sync.Mutex
	jobs    map[string]domain.Job
	reports map[string]domain.Report
}

func NewStore() *Store {
	return &Store{jobs: map[string]domain.Job{}, reports: map[string]domain.Report{}}
}

func (s *Store) Create(_ context.Context, job domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[job.ID]; ok {
		return domain.ErrConflict
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *Store) Get(_ context.Context, id string) (domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return domain.Job{}, domain.ErrNotFound
	}
	return job, nil
}

func (s *Store) List(_ context.Context) ([]domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		out = append(out, job)
	}
	return out, nil
}

func (s *Store) Update(_ context.Context, job domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[job.ID]; !ok {
		return domain.ErrNotFound
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *Store) SaveReport(_ context.Context, jobID string, report domain.Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports[jobID] = report
	return nil
}

func (s *Store) GetReport(_ context.Context, jobID string) (domain.Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report, ok := s.reports[jobID]
	if !ok {
		return domain.Report{}, domain.ErrNotFound
	}
	return report, nil
}
