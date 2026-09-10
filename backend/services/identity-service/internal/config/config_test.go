package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("IDENTITY_GRPC_ADDR", "")
	t.Setenv("IDENTITY_HEALTH_ADDR", "")
	t.Setenv("IDENTITY_ENV", "")
	t.Setenv("APP_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9091" || cfg.HealthAddr != ":8082" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsProductionMemoryStore(t *testing.T) {
	cfg := Config{Environment: "production", TwoFAAddr: "twofa:9092", PasskeyAddr: "passkey:9093"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing database error")
	}
}

func TestValidateRejectsProductionMissingAuthDependencies(t *testing.T) {
	cfg := Config{Environment: "production", DatabaseURL: "postgres://db"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing auth dependency error")
	}
}

func TestValidateAcceptsProductionConfig(t *testing.T) {
	cfg := Config{
		Environment: "production", DatabaseURL: "postgres://db", TwoFAAddr: "twofa:9092", PasskeyAddr: "passkey:9093",
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
