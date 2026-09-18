package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/inbound-service/internal/config"
	"order-fill/backend/services/inbound-service/internal/migrate"
	"order-fill/backend/services/inbound-service/internal/service/inbound"
	"order-fill/backend/services/inbound-service/internal/storage/objectstore"
	"order-fill/backend/services/inbound-service/internal/storage/postgres"
	"order-fill/backend/services/inbound-service/internal/transport/grpcapi"
)

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("inbound store is required")
	}
	objects, err := objectstore.New(
		cfg.InboundS3.Endpoint,
		cfg.InboundS3.AccessKey,
		cfg.InboundS3.SecretKey,
		cfg.InboundS3.Bucket,
		cfg.InboundS3.UseSSL,
	)
	if err != nil {
		return err
	}
	if err := objects.EnsureBucket(ctx); err != nil {
		return err
	}

	pool, err := postgres.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrate.Up(ctx, pool); err != nil {
		return err
	}
	log.Info("inbound-service using postgres")

	svc := inbound.New(postgres.New(pool), objects)
	health := http.NewServeMux()
	health.Handle("GET /healthz", healthz.Live())
	health.Handle("GET /readyz", healthz.Ready(pool.Ping))
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(svc, cfg.WorkerToken)), health)
}
