package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-fill/backend/pkg/healthz"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("GATEWAY_ADDR", "")
	t.Setenv("GATEWAY_ENV", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("API_ALLOWED_ORIGINS", "")
	t.Setenv("SESSION_COOKIE_SECURE", "")
	cfg := Load()
	if cfg.Addr != ":8080" || cfg.Environment != "local" || cfg.CookieSecure ||
		cfg.AllowedOrigins != "http://127.0.0.1:3200,http://localhost:3200" || cfg.BrandGRPC != "127.0.0.1:9098" {
		t.Fatalf("%+v", cfg)
	}
	if cfg.InboundWebhookRPS != 10 || cfg.InboundWebhookBurst != 20 {
		t.Fatalf("webhook limiter defaults: rps=%v burst=%d", cfg.InboundWebhookRPS, cfg.InboundWebhookBurst)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCookieSecureDefaultsOnOutsideLocal(t *testing.T) {
	t.Setenv("GATEWAY_ENV", "production")
	t.Setenv("API_ALLOWED_ORIGINS", "https://orderfill.example.com")
	t.Setenv("SESSION_COOKIE_SECURE", "")
	t.Setenv("GRPC_TLS_MODE", "mtls")
	t.Setenv("GRPC_TLS_CERT_FILE", "cert.pem")
	t.Setenv("GRPC_TLS_KEY_FILE", "key.pem")
	t.Setenv("GRPC_TLS_CA_FILE", "ca.pem")
	t.Setenv("INBOUND_WEBHOOK_TOKEN", "production-inbound-webhook-token")
	t.Setenv("WORKER_TOKEN", "production-worker-token-123456")
	cfg := Load()
	if !cfg.CookieSecure {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsProductionWildcardOrigins(t *testing.T) {
	cfg := Config{Environment: "production", AllowedOrigins: "*", CookieSecure: true}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected wildcard origin error")
	}
}

func TestValidateRejectsProductionHTTPOrigins(t *testing.T) {
	cfg := Config{Environment: "production", AllowedOrigins: "http://orderfill.example.com", CookieSecure: true}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected http origin error")
	}
}

func TestValidateRejectsProductionInsecureCookie(t *testing.T) {
	cfg := Config{Environment: "production", AllowedOrigins: "https://orderfill.example.com", CookieSecure: false}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected insecure cookie error")
	}
}

func TestValidateRejectsDefaultInboundWebhookToken(t *testing.T) {
	cfg := Config{
		Environment: "production", AllowedOrigins: "https://orderfill.example.com", CookieSecure: true,
		InboundWebhook: "local-dev-inbound-webhook-token", WorkerToken: "production-worker-token-123456",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected inbound webhook token error")
	}
}

func TestValidateRejectsShortWorkerToken(t *testing.T) {
	cfg := Config{
		Environment: "production", AllowedOrigins: "https://orderfill.example.com", CookieSecure: true,
		InboundWebhook: "production-inbound-webhook-token", WorkerToken: "short",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected worker token error")
	}
}

func TestLoadWebhookLimiterFromEnv(t *testing.T) {
	t.Setenv("INBOUND_WEBHOOK_RPS", "5")
	t.Setenv("INBOUND_WEBHOOK_BURST", "8")
	cfg := Load()
	if cfg.InboundWebhookRPS != 5 || cfg.InboundWebhookBurst != 8 {
		t.Fatalf("rps=%v burst=%d", cfg.InboundWebhookRPS, cfg.InboundWebhookBurst)
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
