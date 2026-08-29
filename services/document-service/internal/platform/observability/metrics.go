package observability

import "sync"

type Metrics struct {
	mu                   sync.RWMutex
	jobsCompleted        int64
	jobsFailed           int64
	processingDurationMS int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) AddJobCompleted(durationMS int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobsCompleted++
	m.processingDurationMS += durationMS
}

func (m *Metrics) AddJobFailed(durationMS int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobsFailed++
	m.processingDurationMS += durationMS
}

func (m *Metrics) Snapshot() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]int64{
		"jobs_completed":         m.jobsCompleted,
		"jobs_failed":            m.jobsFailed,
		"processing_duration_ms": m.processingDurationMS,
	}
}
