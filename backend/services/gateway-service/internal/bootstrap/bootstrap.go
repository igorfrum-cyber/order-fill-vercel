package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"order-fill/backend/services/gateway-service/internal/clients"
	"order-fill/backend/services/gateway-service/internal/config"
	"order-fill/backend/services/gateway-service/internal/transport/httpapi"
)

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	c, err := clients.Dial(ctx, cfg)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(cfg, c),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("gateway listening", "addr", cfg.Addr)
		errc <- httpSrv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
}
