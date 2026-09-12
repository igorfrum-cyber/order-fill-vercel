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

var consumeScript = redis.NewScript(`
local n = redis.call("INCR", KEYS[1])
if n == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
if n > tonumber(ARGV[2]) then
  return 0
end
return 1
`)

// Limiter counts TOTP attempts per user.
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

func (l *Limiter) Allow(ctx context.Context, key string) bool {
	if l == nil {
		return false
	}
	if l.client != nil {
		allowed, err := consumeScript.Run(ctx, l.client, []string{keyPrefix + key}, int(l.window.Seconds()), l.max).Int()
		if err != nil {
			return false
		}
		return allowed == 1
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
	l.hits[key] = append(kept, now)
	return true
}

func (l *Limiter) Clear(ctx context.Context, key string) {
	if l == nil {
		return
	}
	if l.client != nil {
		_ = l.client.Del(ctx, keyPrefix+key).Err()
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
