package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"order-fill/backend/pkg/logger"
	"order-fill/backend/services/identity-service/internal/bootstrap"
	"order-fill/backend/services/identity-service/internal/config"
)

func main() {
	log := logger.New()
	if err := run(log); err != nil {
		log.Error("identity stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return bootstrap.Run(ctx, config.Load(), log)
}
