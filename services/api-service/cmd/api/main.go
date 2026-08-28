package main

import (
	"log/slog"
	"net/http"
	"os"

	httpapi "order-fill/services/api-service/internal/http"
)

func main() {
	addr := getenv("API_ADDR", ":8080")

	slog.Info("starting api-service", "service", "api-service", "addr", addr)
	if err := http.ListenAndServe(addr, httpapi.NewRouter()); err != nil {
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
