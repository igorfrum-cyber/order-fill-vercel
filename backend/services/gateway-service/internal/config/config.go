package config

import "os"

type Config struct {
	Addr           string
	Environment    string
	IdentityGRPC   string
	IdentityHTTP   string
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
	return Config{
		Addr:           getenv("GATEWAY_ADDR", ":8080"),
		Environment:    getenv("GATEWAY_ENV", "local"),
		IdentityGRPC:   getenv("IDENTITY_GRPC_ADDR", "127.0.0.1:9091"),
		IdentityHTTP:   getenv("IDENTITY_HTTP_ADDR", "http://127.0.0.1:8082"),
		TwoFAGRPC:      getenv("TWOFA_GRPC_ADDR", "127.0.0.1:9092"),
		PasskeyGRPC:    getenv("PASSKEY_GRPC_ADDR", "127.0.0.1:9093"),
		JobGRPC:        getenv("JOB_GRPC_ADDR", "127.0.0.1:9094"),
		FileGRPC:       getenv("FILE_GRPC_ADDR", "127.0.0.1:9095"),
		AuditGRPC:      getenv("AUDIT_GRPC_ADDR", "127.0.0.1:9100"),
		WorkerHealth:   getenv("WORKER_HEALTH_URL", "http://127.0.0.1:8092/healthz"),
		FileHealth:     getenv("FILE_HEALTH_URL", "http://127.0.0.1:8086/healthz"),
		PostgresAddr:   getenv("POSTGRES_ADDR", "127.0.0.1:5432"),
		RedisAddr:      getenv("REDIS_ADDR", "127.0.0.1:6379"),
		AllowedOrigins: getenv("API_ALLOWED_ORIGINS", "*"),
		CookieSecure:   getenv("SESSION_COOKIE_SECURE", "false") == "true",
		CookieDomain:   getenv("SESSION_COOKIE_DOMAIN", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
