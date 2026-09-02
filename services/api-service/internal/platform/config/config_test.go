package config

import "testing"

func TestLoadAllowsWildcardOriginsInLocal(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "local")
	t.Setenv("API_ALLOWED_ORIGINS", "*")
	t.Setenv("SESSION_COOKIE_SECURE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("local mode should allow wildcard origins: %v", err)
	}
	if cfg.Environment != "local" {
		t.Fatalf("expected environment local, got %q", cfg.Environment)
	}
	if cfg.AllowedOrigins != "*" {
		t.Fatalf("expected wildcard origins, got %q", cfg.AllowedOrigins)
	}
}

func TestLoadRejectsWildcardOriginsInProduction(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_ALLOWED_ORIGINS", "*")
	t.Setenv("SESSION_COOKIE_SECURE", "true")

	if _, err := Load(); err == nil {
		t.Fatal("production mode must reject wildcard origins")
	}
}

func TestLoadRequiresSecureCookieInProduction(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_ALLOWED_ORIGINS", "https://app.example")
	t.Setenv("SESSION_COOKIE_SECURE", "false")

	if _, err := Load(); err == nil {
		t.Fatal("production mode must require SESSION_COOKIE_SECURE=true")
	}
}

func TestLoadReadsWebAuthnSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("WEBAUTHN_RP_ID", "example.com")
	t.Setenv("WEBAUTHN_RP_DISPLAY_NAME", "Order Fill")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WebAuthnRPID != "example.com" || cfg.WebAuthnRPName != "Order Fill" {
		t.Fatalf("got %#v", cfg)
	}
}

func TestLoadRequiresWebAuthnRPIDInProduction(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_ALLOWED_ORIGINS", "https://example.com")
	t.Setenv("SESSION_COOKIE_SECURE", "true")

	if _, err := Load(); err == nil {
		t.Fatal("production mode must require WEBAUTHN_RP_ID")
	}
}

func TestLoadDefaultsCookieDomainToWebAuthnRPIDInProduction(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_ALLOWED_ORIGINS", "https://example.com")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	t.Setenv("WEBAUTHN_RP_ID", "example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CookieDomain != "example.com" {
		t.Fatalf("cookie domain %q", cfg.CookieDomain)
	}
}

func TestLoadRejectsIPWebAuthnRPID(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("WEBAUTHN_RP_ID", "192.168.31.108")
	if _, err := Load(); err == nil {
		t.Fatal("WEBAUTHN_RP_ID must not be an IP address")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://order_fill:order_fill@localhost:5432/order_fill?sslmode=disable")
	t.Setenv("QUEUE_URL", "redis://localhost:6379/0")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "minioadmin")
	t.Setenv("S3_SECRET_KEY", "minioadmin")
}
