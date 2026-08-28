package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	addr := getenv("API_ADDR", ":8080")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	slog.Info("starting api-service", "service", "api-service", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("api-service stopped", "service", "api-service", "error", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
