package usecase

import (
	"context"
	"math"
	"sync"
	"time"

	"order-fill/backend/services/document-service/internal/app/port"
)

type jobProgress struct {
	jobs    port.JobStore
	jobID   string
	now     port.Clock
	mu      sync.Mutex
	lastAt  time.Time
	last    float64
	message string
}

func newJobProgress(jobs port.JobStore, jobID string, now port.Clock) *jobProgress {
	return &jobProgress{jobs: jobs, jobID: jobID, now: now, last: -1}
}

func (p *jobProgress) Set(ctx context.Context, fraction float64, message string) {
	if p == nil || p.jobs == nil {
		return
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	if p.last >= 0 &&
		message == p.message &&
		math.Abs(fraction-p.last) < 0.01 &&
		now.Sub(p.lastAt) < 120*time.Millisecond &&
		fraction < 0.999 {
		return
	}
	p.last = fraction
	p.message = message
	p.lastAt = now
	_ = p.jobs.SetProgress(ctx, p.jobID, fraction, message, now)
}
