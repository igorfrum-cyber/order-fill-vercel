package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MATCHING_GRPC_ADDR", "")
	t.Setenv("MATCHING_HEALTH_ADDR", "")
	t.Setenv("MATCHING_ENV", "")
	t.Setenv("APP_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9097" || cfg.HealthAddr != ":8088" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadUsesAppEnv(t *testing.T) {
	t.Setenv("MATCHING_ENV", "")
	t.Setenv("APP_ENV", "production")
	cfg := Load()
	if cfg.Environment != "production" {
		t.Fatalf("%+v", cfg)
	}
}

func TestValidateRejectsInsecureOutsideLocal(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "insecure")
	if err := (Config{Environment: "production"}).Validate(); err == nil {
		t.Fatal("expected tls fail-fast")
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
