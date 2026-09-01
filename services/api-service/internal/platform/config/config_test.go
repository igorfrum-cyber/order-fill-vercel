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

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://order_fill:order_fill@localhost:5432/order_fill?sslmode=disable")
	t.Setenv("QUEUE_URL", "redis://localhost:6379/0")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "minioadmin")
	t.Setenv("S3_SECRET_KEY", "minioadmin")
}
