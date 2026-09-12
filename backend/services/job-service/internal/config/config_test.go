package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JOB_GRPC_ADDR", "")
	t.Setenv("JOB_HEALTH_ADDR", "")
	t.Setenv("JOB_ENV", "")
	t.Setenv("APP_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9094" || cfg.HealthAddr != ":8085" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsProductionMemoryStore(t *testing.T) {
	cfg := Config{Environment: "production", QueueURL: "redis://redis:6379/0", FileAddr: "file:9095", IdentityAddr: "identity:9091"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing database error")
	}
}

func TestValidateRejectsProductionMissingQueue(t *testing.T) {
	cfg := Config{Environment: "production", DatabaseURL: "postgres://db", FileAddr: "file:9095", IdentityAddr: "identity:9091"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing queue error")
	}
}

func TestValidateAcceptsProductionConfig(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "mtls")
	t.Setenv("GRPC_TLS_CERT_FILE", "cert.pem")
	t.Setenv("GRPC_TLS_KEY_FILE", "key.pem")
	t.Setenv("GRPC_TLS_CA_FILE", "ca.pem")
	cfg := Config{
		Environment: "production", DatabaseURL: "postgres://user:secret@db/order_fill?sslmode=require", QueueURL: "redis://:secret@redis:6379/0",
		FileAddr: "file:9095", IdentityAddr: "identity:9091", WorkerToken: "production-worker-token",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsProductionDefaultWorkerToken(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "mtls")
	cfg := Config{
		Environment: "production", DatabaseURL: "postgres://user:secret@db/order_fill?sslmode=require", QueueURL: "redis://:secret@redis:6379/0",
		FileAddr: "file:9095", IdentityAddr: "identity:9091", WorkerToken: "local-dev-worker-token",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected default worker token error")
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
