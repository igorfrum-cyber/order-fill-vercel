package bootstrap

import (
	"context"
	"encoding/json"
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
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var store identityDB = memory.NewStore()
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
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	mux.HandleFunc("GET /public/companies/{slug}", func(w http.ResponseWriter, r *http.Request) {
		company, err := companySvc.PublicBySlug(r.Context(), r.PathValue("slug"))
		if err != nil {
			http.Error(w, `{"code":"not_found","message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         company.ID,
			"name":       company.Name,
			"login_slug": company.LoginSlug,
			"has_logo":   company.HasLogo(),
		})
	})
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(grpcapi.NewServer(authSvc, userSvc, companySvc)), mux)
}
