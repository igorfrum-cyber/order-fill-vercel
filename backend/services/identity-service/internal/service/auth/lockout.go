package auth

import (
	"strings"
	"sync"
	"time"
)

const (
	loginMax    = 5
	loginWindow = 15 * time.Minute
)

type loginGuard struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func (g *loginGuard) allow(login string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(login))
	kept := g.kept(key, now)
	g.hits[key] = kept
	return len(kept) < loginMax
}

func (g *loginGuard) fail(login string, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(login))
	g.hits[key] = append(g.kept(key, now), now)
}

func (g *loginGuard) clear(login string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.hits != nil {
		delete(g.hits, strings.ToLower(strings.TrimSpace(login)))
	}
}

func (g *loginGuard) kept(key string, now time.Time) []time.Time {
	if g.hits == nil {
		g.hits = map[string][]time.Time{}
	}
	cutoff := now.Add(-loginWindow)
	kept := g.hits[key][:0]
	for _, at := range g.hits[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	return kept
}
