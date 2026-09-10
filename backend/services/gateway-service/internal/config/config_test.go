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
		cfg.AllowedOrigins != "http://127.0.0.1:3200,http://localhost:3200" {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCookieSecureDefaultsOnOutsideLocal(t *testing.T) {
	t.Setenv("GATEWAY_ENV", "production")
	t.Setenv("API_ALLOWED_ORIGINS", "https://orderfill.example.com")
	t.Setenv("SESSION_COOKIE_SECURE", "")
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

func TestHealthHandler(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	healthz.Live().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}
