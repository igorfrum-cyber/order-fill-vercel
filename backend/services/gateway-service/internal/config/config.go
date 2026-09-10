package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Addr           string
	Environment    string
	IdentityGRPC   string
	TwoFAGRPC      string
	PasskeyGRPC    string
	JobGRPC        string
	FileGRPC       string
	AuditGRPC      string
	WorkerHealth   string
	FileHealth     string
	PostgresAddr   string
	RedisAddr      string
	AllowedOrigins string
	CookieSecure   bool
	CookieDomain   string
}

func Load() Config {
	env := getenv("GATEWAY_ENV", getenv("APP_ENV", "local"))
	return Config{
		Addr:           getenv("GATEWAY_ADDR", ":8080"),
		Environment:    env,
		IdentityGRPC:   getenv("IDENTITY_GRPC_ADDR", "127.0.0.1:9091"),
		TwoFAGRPC:      getenv("TWOFA_GRPC_ADDR", "127.0.0.1:9092"),
		PasskeyGRPC:    getenv("PASSKEY_GRPC_ADDR", "127.0.0.1:9093"),
		JobGRPC:        getenv("JOB_GRPC_ADDR", "127.0.0.1:9094"),
		FileGRPC:       getenv("FILE_GRPC_ADDR", "127.0.0.1:9095"),
		AuditGRPC:      getenv("AUDIT_GRPC_ADDR", "127.0.0.1:9100"),
		WorkerHealth:   getenv("WORKER_HEALTH_URL", "http://127.0.0.1:8092/healthz"),
		FileHealth:     getenv("FILE_HEALTH_URL", "http://127.0.0.1:8086/healthz"),
		PostgresAddr:   getenv("POSTGRES_ADDR", "127.0.0.1:5432"),
		RedisAddr:      getenv("REDIS_ADDR", "127.0.0.1:6379"),
		AllowedOrigins: getenv("API_ALLOWED_ORIGINS", defaultAllowedOrigins(env)),
		CookieSecure:   cookieSecure(env),
		CookieDomain:   getenv("SESSION_COOKIE_DOMAIN", ""),
	}
}

func (c Config) Validate() error {
	if localEnv(c.Environment) {
		return nil
	}
	if !c.CookieSecure {
		return fmt.Errorf("SESSION_COOKIE_SECURE must be true outside local environment")
	}
	if strings.TrimSpace(c.AllowedOrigins) == "" {
		return fmt.Errorf("API_ALLOWED_ORIGINS is required outside local environment")
	}
	for _, origin := range splitOrigins(c.AllowedOrigins) {
		if origin == "*" {
			return fmt.Errorf("API_ALLOWED_ORIGINS must not contain * outside local environment")
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("API_ALLOWED_ORIGINS contains invalid origin %q", origin)
		}
		if parsed.Scheme != "https" {
			return fmt.Errorf("API_ALLOWED_ORIGINS must use https outside local environment: %q", origin)
		}
	}
	return nil
}

func cookieSecure(env string) bool {
	if raw := os.Getenv("SESSION_COOKIE_SECURE"); raw != "" {
		return raw == "true"
	}
	return env != "local"
}

func defaultAllowedOrigins(env string) string {
	if localEnv(env) {
		return "http://127.0.0.1:3200,http://localhost:3200"
	}
	return ""
}

func localEnv(env string) bool {
	return strings.EqualFold(strings.TrimSpace(env), "local") || strings.TrimSpace(env) == ""
}

func splitOrigins(value string) []string {
	origins := make([]string, 0)
	for _, entry := range strings.Split(value, ",") {
		trimmed := strings.TrimRight(strings.TrimSpace(entry), "/")
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
