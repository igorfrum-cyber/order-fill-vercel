package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultWindow = 15 * time.Minute
	DefaultMax    = 5
	keyPrefix     = "twofa:fail:"
)

// Limiter counts failed TOTP attempts per user.
type Limiter struct {
	mu     sync.Mutex
	window time.Duration
	max    int
	hits   map[string][]time.Time
	now    func() time.Time
	client *redis.Client
}

func New(now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{window: DefaultWindow, max: DefaultMax, hits: map[string][]time.Time{}, now: now}
}

func Open(redisURL string, now func() time.Time) (*Limiter, error) {
	l := New(now)
	if redisURL == "" {
		return l, nil
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	l.client = redis.NewClient(opt)
	return l, nil
}

func NewRedis(now func() time.Time) *Limiter {
	return New(now)
}

func (l *Limiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	if l.client != nil {
		n, err := l.client.Get(context.Background(), keyPrefix+key).Int()
		if err == redis.Nil {
			return true
		}
		if err != nil {
			return true
		}
		return n < l.max
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)
	kept := l.hits[key][:0]
	for _, at := range l.hits[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= l.max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = kept
	return true
}

func (l *Limiter) Fail(key string) {
	if l == nil {
		return
	}
	if l.client != nil {
		pipe := l.client.TxPipeline()
		k := keyPrefix + key
		pipe.Incr(context.Background(), k)
		pipe.Expire(context.Background(), k, l.window)
		_, _ = pipe.Exec(context.Background())
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hits[key] = append(l.hits[key], l.now())
}

func (l *Limiter) Clear(key string) {
	if l == nil {
		return
	}
	if l.client != nil {
		_ = l.client.Del(context.Background(), keyPrefix+key).Err()
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
