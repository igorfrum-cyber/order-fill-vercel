package bootstrap

import (
	"context"
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

	var store inbound.Store
	var readyCheck func(context.Context) error
	if cfg.DatabaseURL != "" {
		pool, err := postgres.OpenPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := migrate.Up(ctx, pool); err != nil {
			return err
		}
		store = postgres.New(pool)
		readyCheck = pool.Ping
		log.Info("inbound-service using postgres")
	} else {
		log.Info("inbound-service: no DATABASE_URL, store is nil")
	}

	if store == nil {
		return nil
	}

	svc := inbound.New(store, objects, cfg.InboundS3.Bucket)
	health := http.NewServeMux()
	health.Handle("GET /healthz", healthz.Live())
	health.Handle("GET /readyz", healthz.Ready(readyCheck))
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(svc, cfg.WorkerToken)), health)
}
