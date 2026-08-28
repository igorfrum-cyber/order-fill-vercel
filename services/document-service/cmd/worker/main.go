package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"order-fill/services/document-service/internal/observability"
)

func main() {
	addr := getenv("WORKER_HEALTH_ADDR", ":8081")
	metrics := observability.NewMetrics()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metrics.Snapshot())
	})

	slog.Info("starting document-service worker", "service", "document-service", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("document-service stopped", "service", "document-service", "error", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
