package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/job-service/internal/clients/files"
	"order-fill/backend/services/job-service/internal/clients/identity"
	"order-fill/backend/services/job-service/internal/config"
	"order-fill/backend/services/job-service/internal/migrate"
	"order-fill/backend/services/job-service/internal/queue"
	"order-fill/backend/services/job-service/internal/service/jobs"
	"order-fill/backend/services/job-service/internal/storage/memory"
	"order-fill/backend/services/job-service/internal/storage/postgres"
	"order-fill/backend/services/job-service/internal/transport/grpcapi"
)

func HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var store jobs.Store = memory.NewStore()
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
		log.Info("job-service using postgres")
	}
	var publisher jobs.Publisher = queue.NewRedis()
	if cfg.QueueURL != "" {
		stream, err := queue.NewStream(cfg.QueueURL, "")
		if err != nil {
			return err
		}
		publisher = stream
	}
	var catalog jobs.Files
	if cfg.FileAddr != "" {
		client, err := files.Dial(ctx, cfg.FileAddr)
		if err != nil {
			return err
		}
		catalog = client
	}
	var companies jobs.Companies
	if cfg.IdentityAddr != "" {
		client, err := identity.Dial(ctx, cfg.IdentityAddr)
		if err != nil {
			return err
		}
		companies = client
	}
	svc := jobs.New(store, catalog, companies, publisher, nil)
	log.Info("job-service listening", "grpc", cfg.GRPCAddr)
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(svc)), HealthHandler())
}
