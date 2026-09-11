package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AUDIT_GRPC_ADDR", "")
	t.Setenv("AUDIT_HEALTH_ADDR", "")
	t.Setenv("AUDIT_ENV", "")
	t.Setenv("APP_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9100" || cfg.HealthAddr != ":8091" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsProductionMemoryStore(t *testing.T) {
	cfg := Config{Environment: "production"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing database error")
	}
}

func TestValidateAcceptsProductionConfig(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "tls")
	cfg := Config{Environment: "production", DatabaseURL: "postgres://user:secret@db/order_fill?sslmode=require"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
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
