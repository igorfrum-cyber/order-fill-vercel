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
	auth := usecase.NewAuth(repository, uuid.NewString, now)
	admin := usecase.NewAdmin(repository, uuid.NewString, now).WithFiles(storage)

	invite, created, err := auth.Bootstrap(ctx, settings.BootstrapAdminLogin)
	if err != nil {
		return err
	}
	if created {
		logger.Info("bootstrap admin invite created",
			"service", "api-service",
			"event", "bootstrap_admin",
			"login", settings.BootstrapAdminLogin,
			"invite_url", "/invite/"+invite,
		)
	}

	router := httpapi.NewRouter(httpapi.Config{
		CreateJob:       usecase.NewCreateJob(repository, storage, publisher, uuid.NewString, now, logger, metrics),
		GetJob:          usecase.NewGetJob(repository),
		GetReport:       usecase.NewGetReport(repository),
		ListFiles:       usecase.NewListFiles(repository),
		DownloadFile:    usecase.NewDownloadFile(repository, storage),
		DownloadArchive: usecase.NewDownloadArchive(repository, storage),
		SubmitEdits:     usecase.NewSubmitEdits(repository, publisher, now, logger),
		Preview:         usecase.NewPreviewReader(repository, storage),
		ListJobs:        usecase.NewListJobs(repository),
		Auth:            auth,
		Admin:           admin,
		Reset:           auth,
		Metrics:         metrics,
		AllowedOrigins:  httpapi.ParseAllowedOrigins(settings.AllowedOrigins),
		MaxUploadBytes:  settings.MaxUploadBytes,
		CookieSecure:    settings.CookieSecure,
		LoginLimiter:    httpapi.NewLimiter(15*time.Minute, 10),
		CreateLimiter:   httpapi.NewLimiter(time.Hour, 30),
	})

	server := &http.Server{
		Addr:              settings.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
