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
	c, err := clients.Dial(ctx, cfg)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{Addr: cfg.Addr, Handler: httpapi.New(cfg, c), ReadHeaderTimeout: 10 * time.Second}
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
