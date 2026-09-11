package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PASSKEY_GRPC_ADDR", "")
	t.Setenv("PASSKEY_HEALTH_ADDR", "")
	t.Setenv("PASSKEY_ENV", "")
	t.Setenv("APP_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9093" || cfg.HealthAddr != ":8084" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsProductionWithoutRPID(t *testing.T) {
	cfg := Config{Environment: "production", DatabaseURL: "postgres://db", RedisURL: "redis://redis:6379/0"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing RP ID error")
	}
}

func TestValidateRejectsProductionMemoryDependencies(t *testing.T) {
	cfg := Config{Environment: "production", RPID: "orderfill.example.com"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing dependency error")
	}
}

func TestValidateAcceptsProductionConfig(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "tls")
	cfg := Config{
		Environment: "production", DatabaseURL: "postgres://user:secret@db/order_fill?sslmode=require", RedisURL: "redis://:secret@redis:6379/0", RPID: "orderfill.example.com",
	}
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
