package observability

import "sync"

type Metrics struct {
	mu          sync.RWMutex
	jobsCreated int64
	jobsFailed  int64
	bytesStored int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) AddJobCreated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobsCreated++
}

func (m *Metrics) AddJobFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobsFailed++
}

func (m *Metrics) AddBytesStored(value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bytesStored += value
}

func (m *Metrics) Snapshot() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]int64{
		"jobs_created": m.jobsCreated,
		"jobs_failed":  m.jobsFailed,
		"bytes_stored": m.bytesStored,
	}
}
