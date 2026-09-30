package ratelimit

import (
	"testing"
	"time"
)

func TestAllowLocksAfterMaxAttempts(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	limiter := New(func() time.Time { return now })
	for i := 0; i < DefaultMax; i++ {
		if !limiter.Allow(t.Context(), "u1") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if limiter.Allow(t.Context(), "u1") {
		t.Fatal("attempt after max should be locked")
	}
}

func TestAllowNilFailsClosed(t *testing.T) {
	t.Parallel()
	var limiter *Limiter
	if limiter.Allow(t.Context(), "u1") {
		t.Fatal("nil limiter must deny")
	}
}

func TestAllowExpiresOldAttemptsAndSeparatesKeys(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	limiter := New(func() time.Time { return now })
	for range DefaultMax {
		if !limiter.Allow(t.Context(), "user-a") {
			t.Fatal("attempt inside limit denied")
		}
	}
	if !limiter.Allow(t.Context(), "user-b") {
		t.Fatal("another key should have an independent limit")
	}
	now = now.Add(DefaultWindow)
	if !limiter.Allow(t.Context(), "user-a") {
		t.Fatal("attempts at the window boundary should expire")
	}
}

func TestClearResetsOneKey(t *testing.T) {
	t.Parallel()

	limiter := New(func() time.Time { return time.Unix(1_700_000_000, 0) })
	for range DefaultMax {
		if !limiter.Allow(t.Context(), "user-a") {
			t.Fatal("attempt inside limit denied")
		}
	}
	if limiter.Allow(t.Context(), "user-a") {
		t.Fatal("attempt after limit should be denied")
	}
	limiter.Clear(t.Context(), "user-a")
	if !limiter.Allow(t.Context(), "user-a") {
		t.Fatal("cleared key remained limited")
	}
}

func TestOpenSupportsInMemoryAndRedisConfiguration(t *testing.T) {
	t.Parallel()

	inMemory, err := Open("", nil)
	if err != nil || !inMemory.Allow(t.Context(), "user-a") {
		t.Fatalf("Open(empty) = %v, %v", inMemory, err)
	}
	if _, err := Open("not a redis url", nil); err == nil {
		t.Fatal("Open accepted malformed Redis URL")
	}
	redisBacked, err := Open("redis://localhost:6379/0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if redisBacked.client == nil {
		t.Fatal("Redis URL did not configure a Redis client")
	}
	if err := redisBacked.client.Close(); err != nil {
		t.Fatal(err)
	}
	if NewRedis(nil).client != nil {
		t.Fatal("NewRedis should remain an in-memory limiter constructor")
	}
}

func TestClearOnNilLimiterIsSafe(t *testing.T) {
	t.Parallel()

	var limiter *Limiter
	limiter.Clear(t.Context(), "user-a")
}
