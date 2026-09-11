package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DOCUMENT_GRPC_ADDR", "")
	t.Setenv("DOCUMENT_HEALTH_ADDR", "")
	t.Setenv("DOCUMENT_ENV", "")
	t.Setenv("APP_ENV", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9096" || cfg.HealthAddr != ":8087" || cfg.Environment != "local" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.ValidateAPI(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateWorker(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAPIRejectsProductionMissingDependencies(t *testing.T) {
	cfg := Config{Environment: "production", FileAddr: "file:9095"}
	if err := cfg.ValidateAPI(); err == nil {
		t.Fatal("expected missing brand dependency error")
	}
}

func TestValidateWorkerRejectsProductionMissingDependencies(t *testing.T) {
	cfg := Config{Environment: "production", QueueURL: "redis://redis:6379/0"}
	if err := cfg.ValidateWorker(); err == nil {
		t.Fatal("expected missing worker dependency error")
	}
}

func TestValidateProductionModesAcceptCompleteConfig(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "mtls")
	t.Setenv("GRPC_TLS_CERT_FILE", "cert.pem")
	t.Setenv("GRPC_TLS_KEY_FILE", "key.pem")
	t.Setenv("GRPC_TLS_CA_FILE", "ca.pem")
	cfg := Config{
		Environment: "production", QueueURL: "redis://:secret@redis:6379/0", JobAddr: "job:9094",
		FileAddr: "file:9095", CalculationAddr: "calculation:9099", MatchingAddr: "matching:9097", BrandAddr: "brand:9098",
		WorkerToken: "production-worker-token",
	}
	if err := cfg.ValidateAPI(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateWorker(); err != nil {
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
