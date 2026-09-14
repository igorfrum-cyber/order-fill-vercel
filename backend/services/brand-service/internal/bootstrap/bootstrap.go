package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/brand-service/internal/config"
	"order-fill/backend/services/brand-service/internal/migrate"
	"order-fill/backend/services/brand-service/internal/service/brands"
	"order-fill/backend/services/brand-service/internal/storage/memory"
	"order-fill/backend/services/brand-service/internal/storage/postgres"
	"order-fill/backend/services/brand-service/internal/transport/grpcapi"
)

func HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	var store brands.Store = memory.New()
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
		log.Info("brand-service using postgres")
	}
	health := http.NewServeMux()
	health.Handle("GET /healthz", healthz.Live())
	health.Handle("GET /readyz", healthz.Ready(readyCheck))
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(brands.New(store))), health)
}
