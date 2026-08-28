package jobs

import (
	"context"
	"errors"
	"sync"
)

var ErrJobNotFound = errors.New("job not found")

type MemoryRepository struct {
	mu      sync.RWMutex
	jobs    map[string]Job
	reports map[string]Report
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		jobs:    make(map[string]Job),
		reports: make(map[string]Report),
	}
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

func (r *MemoryRepository) UpdateStatus(_ context.Context, id string, status JobStatus) (Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	job.Status = status
	job.UpdatedAt = job.UpdatedAt.Add(1)
	r.jobs[id] = cloneJob(job)
	return cloneJob(job), nil
}

func (r *MemoryRepository) SaveReport(_ context.Context, report Report) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports[report.JobID] = cloneReport(report)
	return nil
}

func (r *MemoryRepository) Report(_ context.Context, jobID string) (Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	report, ok := r.reports[jobID]
	if !ok {
		return Report{}, ErrJobNotFound
	}
	return cloneReport(report), nil
}

func cloneJob(job Job) Job {
	job.InputFiles = append([]StoredFile(nil), job.InputFiles...)
	job.OutputFiles = append([]OutputFile(nil), job.OutputFiles...)
	return job
}

func cloneReport(report Report) Report {
	report.Rows = append([]ReportRow(nil), report.Rows...)
	return report
}
