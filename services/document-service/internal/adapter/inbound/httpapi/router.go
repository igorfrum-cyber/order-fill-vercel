// Package httpapi exposes the worker's liveness and metrics surface. The worker
// has no public API; this router only exists so the platform can probe it.
package httpapi

import (
	"encoding/json"
	"net/http"
)

// MetricsSnapshotter reports the worker counters, matching observability.Metrics.
type MetricsSnapshotter interface {
	Snapshot() map[string]int64
}

func NewRouter(metrics MetricsSnapshotter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		if metrics == nil {
			writeJSON(w, map[string]int64{})
			return
		}
		writeJSON(w, metrics.Snapshot())
	})
	return mux
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
