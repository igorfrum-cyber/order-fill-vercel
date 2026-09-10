package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	"order-fill/backend/services/identity-service/internal/clients/passkey"
	"order-fill/backend/services/identity-service/internal/clients/twofa"
	"order-fill/backend/services/identity-service/internal/config"
	"order-fill/backend/services/identity-service/internal/migrate"
	"order-fill/backend/services/identity-service/internal/service/auth"
	"order-fill/backend/services/identity-service/internal/service/companies"
	"order-fill/backend/services/identity-service/internal/service/users"
	"order-fill/backend/services/identity-service/internal/storage/memory"
	"order-fill/backend/services/identity-service/internal/storage/postgres"
	"order-fill/backend/services/identity-service/internal/transport/grpcapi"
)

type identityDB interface {
	auth.Store
	users.Store
	companies.Store
}

func HealthHandler() http.Handler {
	return healthHandler(nil)
}

func healthHandler(check func(context.Context) error) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(check))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	var store identityDB = memory.NewStore()
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
		store = postgres.NewStore(pool)
		readyCheck = pool.Ping
		log.Info("identity-service using postgres")
	}
	var totp twofa.Client
	if cfg.TwoFAAddr != "" {
		client, err := twofa.Dial(ctx, cfg.TwoFAAddr)
		if err != nil {
			return err
		}
		totp = client
	}
	var keys passkey.Client
	if cfg.PasskeyAddr != "" {
		client, err := passkey.Dial(ctx, cfg.PasskeyAddr)
		if err != nil {
			return err
		}
		keys = client
	}
	authSvc := auth.New(store, totp, keys, nil)
	userSvc := users.New(store, nil)
	companySvc := companies.New(store, nil)
	invite, created, err := authSvc.Bootstrap(ctx, cfg.BootstrapAdminLogin)
	if err != nil {
		return err
	}
	if created {
		log.Info("bootstrap admin invite", "login", cfg.BootstrapAdminLogin, "invite_url", "/invite/"+invite)
	}
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(authSvc, userSvc, companySvc)), healthHandler(readyCheck))
}
