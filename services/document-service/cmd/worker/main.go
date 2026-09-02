// Command worker is the composition root of document-service: it wires the
// queue consumer to the order-fill use case and the infrastructure adapters.
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

	"golang.org/x/sync/errgroup"

	"order-fill/services/document-service/internal/adapter/inbound/httpapi"
	"order-fill/services/document-service/internal/adapter/inbound/queue"
	"order-fill/services/document-service/internal/adapter/outbound/jobstore"
	"order-fill/services/document-service/internal/adapter/outbound/objectstore"
	"order-fill/services/document-service/internal/adapter/outbound/xlsx"
	"order-fill/services/document-service/internal/app/usecase"
	"order-fill/services/document-service/internal/platform/config"
	"order-fill/services/document-service/internal/platform/observability"
)

const shutdownTimeout = 15 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("document-service stopped", "service", "document-service", "event", "startup_failed", "error", err)
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

	pool, err := jobstore.OpenPool(ctx, settings.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	storage, err := objectstore.NewStore(settings.S3Endpoint, settings.S3AccessKey, settings.S3SecretKey, settings.S3Bucket)
	if err != nil {
		return err
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		return err
	}

	consumer, err := queue.NewConsumer(settings.QueueURL, settings.QueueName, logger)
	if err != nil {
		return err
	}
	defer func() { _ = consumer.Close() }()

	metrics := observability.NewMetrics()
	store := jobstore.NewStore(pool)
	processor := usecase.NewProcessJob(
		xlsx.NewCodec(),
		storage,
		store,
		store,
		func() time.Time { return time.Now().UTC() },
		logger,
		metrics,
	)

	server := &http.Server{
		Addr:              settings.HealthAddr,
		Handler:           httpapi.NewRouter(metrics),
		ReadHeaderTimeout: 10 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()

	consumerErrors := make(chan error, 1)
	go func() {
		logger.Info("starting document-service",
			"service", "document-service",
			"event", "worker_started",
			"addr", settings.HealthAddr,
			"queue", settings.QueueName,
			"worker_concurrency", settings.WorkerConcurrency,
		)
		consumerErrors <- runConsumerPool(ctx, consumer, settings.WorkerConcurrency, processor.Handle)
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case err := <-consumerErrors:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	logger.Info("stopping document-service", "service", "document-service", "event", "worker_stopping")
	return server.Shutdown(shutdownCtx)
}

type consumerRunner interface {
	Run(context.Context, queue.Handler) error
}

func runConsumerPool(ctx context.Context, consumer consumerRunner, concurrency int, handle queue.Handler) error {
	if concurrency < 1 {
		concurrency = 1
	}
	group, groupCtx := errgroup.WithContext(ctx)
	for i := 0; i < concurrency; i++ {
		group.Go(func() error {
			return consumer.Run(groupCtx, handle)
		})
	}
	return group.Wait()
}
