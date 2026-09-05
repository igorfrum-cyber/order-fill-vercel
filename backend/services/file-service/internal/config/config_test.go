package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("FILE_GRPC_ADDR", "")
	t.Setenv("FILE_HEALTH_ADDR", "")
	t.Setenv("FILE_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9095" || cfg.HealthAddr != ":8086" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
}

func TestHealthHandler(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	healthz.Live().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}
