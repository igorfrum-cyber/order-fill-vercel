package metrics

import (
	"sync"
	"testing"
)

func TestHTTPIncAndSnapshot(t *testing.T) {
	t.Parallel()

	metrics := NewHTTP()
	if got := metrics.Snapshot(); len(got) != 0 {
		t.Fatalf("initial snapshot = %v, want empty", got)
	}

	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 100 {
				metrics.Inc("requests")
			}
		})
	}
	workers.Wait()

	snapshot := metrics.Snapshot()
	if got := snapshot["requests"]; got != 800 {
		t.Fatalf("request count = %d, want 800", got)
	}
	snapshot["requests"] = -1
	if got := metrics.Snapshot()["requests"]; got != 800 {
		t.Fatalf("mutating snapshot changed live count to %d", got)
	}
}
