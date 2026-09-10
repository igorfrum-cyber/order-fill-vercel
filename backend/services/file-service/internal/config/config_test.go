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
	t.Setenv("APP_ENV", "")
	t.Setenv("FILE_S3_USE_SSL", "")
	cfg := Load()
	if cfg.GRPCAddr != ":9095" || cfg.HealthAddr != ":8086" || cfg.Environment != "local" || cfg.S3UseSSL {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestProductionDefaultsS3SSL(t *testing.T) {
	t.Setenv("FILE_ENV", "production")
	t.Setenv("FILE_S3_USE_SSL", "")
	cfg := Load()
	if !cfg.S3UseSSL {
		t.Fatalf("%+v", cfg)
	}
}

func TestValidateRejectsProductionMemoryStore(t *testing.T) {
	cfg := Config{Environment: "production", S3UseSSL: true, S3AccessKey: "access", S3SecretKey: "secret"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing storage config error")
	}
}

func TestValidateRejectsProductionInsecureS3(t *testing.T) {
	cfg := Config{
		Environment: "production", DatabaseURL: "postgres://db", S3Endpoint: "s3.example.com",
		S3UseSSL: false, S3AccessKey: "access", S3SecretKey: "secret",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected insecure object store error")
	}
}

func TestValidateRejectsProductionDefaultS3Credentials(t *testing.T) {
	cfg := Config{
		Environment: "production", DatabaseURL: "postgres://db", S3Endpoint: "s3.example.com",
		S3UseSSL: true, S3AccessKey: "minioadmin", S3SecretKey: "secret",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected default credentials error")
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
