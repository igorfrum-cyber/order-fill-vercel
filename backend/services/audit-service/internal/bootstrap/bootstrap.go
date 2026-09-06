package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/audit-service/internal/config"
	"order-fill/backend/services/audit-service/internal/migrate"
	"order-fill/backend/services/audit-service/internal/service/audit"
	"order-fill/backend/services/audit-service/internal/storage/memory"
	"order-fill/backend/services/audit-service/internal/storage/postgres"
	"order-fill/backend/services/audit-service/internal/transport/grpcapi"
)

func HealthHandler() http.Handler {
	return healthHandler(nil)
}

func healthHandler(check func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(check))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	var store audit.Store = memory.New()
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
		log.Info("audit-service using postgres")
	}
	svc := audit.New(store, nil)
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(svc)), healthHandler(readyCheck))
}
