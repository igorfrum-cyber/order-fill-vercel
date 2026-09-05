package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/passkey-service/internal/ceremony"
	"order-fill/backend/services/passkey-service/internal/config"
	"order-fill/backend/services/passkey-service/internal/migrate"
	"order-fill/backend/services/passkey-service/internal/service/passkey"
	"order-fill/backend/services/passkey-service/internal/storage/memory"
	"order-fill/backend/services/passkey-service/internal/storage/postgres"
	"order-fill/backend/services/passkey-service/internal/transport/grpcapi"
	"order-fill/backend/services/passkey-service/internal/webauthn"
)

func HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var store passkey.Store = memory.NewStore()
	if cfg.DatabaseURL != "" {
		pool, err := postgres.OpenPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := migrate.Up(ctx, pool); err != nil {
			return err
		}
		store = postgres.NewStore(pool)
		log.Info("passkey-service using postgres")
	}
	challenges, err := ceremony.Open(cfg.RedisURL)
	if err != nil {
		return err
	}
	svc := passkey.New(store, challenges, webauthn.New(cfg.DisplayName, cfg.RPID), nil, nil)
	log.Info("passkey-service listening", "grpc", cfg.GRPCAddr)
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(svc)), HealthHandler())
}
