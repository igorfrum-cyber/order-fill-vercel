package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TWOFA_GRPC_ADDR", "")
	t.Setenv("TWOFA_HEALTH_ADDR", "")
	t.Setenv("TWOFA_ENV", "")
	t.Setenv("APP_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9092" || cfg.HealthAddr != ":8083" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsProductionDefaultMasterKey(t *testing.T) {
	cfg := Config{Environment: "production", DatabaseURL: "postgres://db", RedisURL: "redis://redis:6379/0", MasterKey: "local-dev-twofa-master-key"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected default master key error")
	}
}

func TestValidateRejectsProductionMemoryDependencies(t *testing.T) {
	cfg := Config{Environment: "production", MasterKey: "01234567890123456789012345678901"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing dependency error")
	}
}

func TestValidateAcceptsProductionConfig(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "tls")
	cfg := Config{
		Environment: "production", DatabaseURL: "postgres://user:secret@db/order_fill?sslmode=require", RedisURL: "redis://:secret@redis:6379/0",
		MasterKey: "01234567890123456789012345678901",
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
