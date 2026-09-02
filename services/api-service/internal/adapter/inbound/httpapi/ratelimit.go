package httpapi

import (
	"sync"
	"time"
)

type Limiter struct {
	mu     sync.Mutex
	window time.Duration
	max    int
	hits   map[string][]time.Time
	now    func() time.Time
}

func NewLimiter(window time.Duration, max int) *Limiter {
	return &Limiter{window: window, max: max, hits: map[string][]time.Time{}, now: time.Now}
}

func (l *Limiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)
	kept := make([]time.Time, 0, len(l.hits[key])+1)
	for _, at := range l.hits[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= l.max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
