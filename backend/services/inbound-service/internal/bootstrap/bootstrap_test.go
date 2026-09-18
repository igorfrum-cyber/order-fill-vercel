package bootstrap

import (
	"log/slog"
	"strings"
	"testing"

	"order-fill/backend/services/inbound-service/internal/config"
)

func TestRunRequiresStore(t *testing.T) {
	t.Parallel()
	err := Run(t.Context(), config.Config{
		Environment: "local",
		WorkerToken: "local-dev-worker-token",
	}, slog.New(slog.DiscardHandler))
	if err == nil || !strings.Contains(err.Error(), "inbound store is required") {
		t.Fatalf("empty DatabaseURL must fail, got %v", err)
	}
}
