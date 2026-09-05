package metrics

import (
	"maps"
	"sync"
)

type HTTP struct {
	mu     sync.Mutex
	counts map[string]int64
}

func NewHTTP() *HTTP {
	return &HTTP{counts: map[string]int64{}}
}

func (h *HTTP) Inc(name string) {
	h.mu.Lock()
	h.counts[name]++
	h.mu.Unlock()
}

func (h *HTTP) Snapshot() map[string]int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return maps.Clone(h.counts)
}
