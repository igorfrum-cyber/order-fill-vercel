package bootstrap

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/twofa-service/internal/config"
	"order-fill/backend/services/twofa-service/internal/migrate"
	"order-fill/backend/services/twofa-service/internal/ratelimit"
	"order-fill/backend/services/twofa-service/internal/secret"
	"order-fill/backend/services/twofa-service/internal/service/twofa"
	"order-fill/backend/services/twofa-service/internal/storage/memory"
	"order-fill/backend/services/twofa-service/internal/storage/postgres"
	"order-fill/backend/services/twofa-service/internal/transport/grpcapi"
)

func HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	key := sha256.Sum256([]byte(cfg.MasterKey))
	box, err := secret.NewBox(key[:])
	if err != nil {
		return err
	}
	var store twofa.Store = memory.NewStore()
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
		log.Info("twofa-service using postgres")
	}
	limit, err := ratelimit.Open(cfg.RedisURL, nil)
	if err != nil {
		return err
	}
	svc := twofa.New(store, box, limit, nil)
	log.Info("twofa-service listening", "grpc", cfg.GRPCAddr)
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(svc)), HealthHandler())
}
