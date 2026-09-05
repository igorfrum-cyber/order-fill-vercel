package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/matching-service/internal/config"
	"order-fill/backend/services/matching-service/internal/service/matching"
	"order-fill/backend/services/matching-service/internal/transport/grpcapi"
)

func HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(matching.New())), HealthHandler())
}
