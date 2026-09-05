package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/file-service/internal/config"
	"order-fill/backend/services/file-service/internal/migrate"
	"order-fill/backend/services/file-service/internal/service/files"
	"order-fill/backend/services/file-service/internal/storage/memory"
	"order-fill/backend/services/file-service/internal/storage/objectstore"
	"order-fill/backend/services/file-service/internal/storage/postgres"
	"order-fill/backend/services/file-service/internal/transport/grpcapi"
)

func HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var blobs files.BlobStore = objectstore.NewS3()
	if cfg.S3Endpoint != "" {
		store, err := objectstore.NewMinIO(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3UseSSL)
		if err != nil {
			return err
		}
		if err := store.EnsureBucket(ctx); err != nil {
			return err
		}
		blobs = store
		log.Info("file-service using minio", "endpoint", cfg.S3Endpoint, "bucket", cfg.S3Bucket)
	}
	var meta files.MetaStore = memory.NewMeta()
	if cfg.DatabaseURL != "" {
		pool, err := postgres.OpenPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := migrate.Up(ctx, pool); err != nil {
			return err
		}
		meta = postgres.NewMeta(pool)
		log.Info("file-service using postgres meta")
	}
	svc := files.New(blobs, meta)
	log.Info("file-service listening", "grpc", cfg.GRPCAddr)
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(svc)), HealthHandler())
}
