// Command api is the composition root of api-service: it is the only place
// that knows which concrete adapter implements which port.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"order-fill/services/api-service/internal/adapter/inbound/httpapi"
	"order-fill/services/api-service/internal/adapter/outbound/objectstore"
	"order-fill/services/api-service/internal/adapter/outbound/postgres"
	"order-fill/services/api-service/internal/adapter/outbound/queue"
	"order-fill/services/api-service/internal/app/usecase"
	"order-fill/services/api-service/internal/platform/config"
	"order-fill/services/api-service/internal/platform/observability"
)

const shutdownTimeout = 15 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("api-service stopped", "service", "api-service", "event", "startup_failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	settings, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.OpenPool(ctx, settings.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	repository := postgres.NewRepository(pool)
	if err := repository.Migrate(ctx); err != nil {
		return err
	}

	storage, err := objectstore.NewStore(settings.S3Endpoint, settings.S3AccessKey, settings.S3SecretKey, settings.S3Bucket, false)
	if err != nil {
		return err
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		return err
	}

	publisher, err := queue.NewPublisher(settings.QueueURL, settings.QueueName)
	if err != nil {
		return err
	}
	defer func() { _ = publisher.Close() }()

	metrics := observability.NewMetrics()
	now := func() time.Time { return time.Now().UTC() }

	router := httpapi.NewRouter(httpapi.Config{
		CreateJob:       usecase.NewCreateJob(repository, storage, publisher, uuid.NewString, now, logger, metrics),
		GetJob:          usecase.NewGetJob(repository),
		GetReport:       usecase.NewGetReport(repository),
		ListFiles:       usecase.NewListFiles(repository),
		DownloadFile:    usecase.NewDownloadFile(repository, storage),
		DownloadArchive: usecase.NewDownloadArchive(repository, storage),
		SubmitEdits:     usecase.NewSubmitEdits(repository, publisher, now, logger),
		Preview:         usecase.NewPreviewReader(repository, storage),
		Metrics:         metrics,
		AllowedOrigins:  httpapi.ParseAllowedOrigins(settings.AllowedOrigins),
		MaxUploadBytes:  settings.MaxUploadBytes,
	})

	server := &http.Server{
		Addr:              settings.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("starting api-service",
			"service", "api-service",
			"event", "server_started",
			"addr", settings.Addr,
			"allowed_origins", settings.AllowedOrigins,
		)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		logger.Info("stopping api-service", "service", "api-service", "event", "server_stopping")
		return server.Shutdown(shutdownCtx)
	}
}
