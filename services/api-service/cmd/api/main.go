package main

import (
	"log/slog"
	"net/http"
	"os"

	httpapi "order-fill/services/api-service/internal/http"
	"order-fill/services/api-service/internal/jobs"
	"order-fill/services/api-service/internal/observability"
	"order-fill/services/api-service/internal/storage"
)

func main() {
	addr := getenv("API_ADDR", ":8080")
	repository := jobs.NewMemoryRepository()
	queue := jobs.NewMemoryQueue()
	objectStorage := storage.NewMemoryObjectStorage()
	metrics := observability.NewMetrics()
	jobService := jobs.NewService(jobs.ServiceConfig{
		Repository: repository,
		Storage:    objectStorage,
		Queue:      queue,
		Logger:     slog.Default(),
		Metrics:    metrics,
	})

	slog.Info("starting api-service", "service", "api-service", "addr", addr)
	if err := http.ListenAndServe(addr, httpapi.NewRouter(httpapi.WithJobService(jobService), httpapi.WithMetrics(metrics))); err != nil {
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
