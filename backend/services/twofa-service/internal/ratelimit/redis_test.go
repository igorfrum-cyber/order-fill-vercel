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
