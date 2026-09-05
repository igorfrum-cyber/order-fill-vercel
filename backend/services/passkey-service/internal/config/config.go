package config

import "os"

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
	DatabaseURL string
	RedisURL    string
	RPID        string
	DisplayName string
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("PASSKEY_GRPC_ADDR", ":9093"),
		HealthAddr:  getenv("PASSKEY_HEALTH_ADDR", ":8084"),
		Environment: getenv("PASSKEY_ENV", "local"),
		DatabaseURL: getenv("DATABASE_URL", ""),
		RedisURL:    getenv("QUEUE_URL", getenv("REDIS_URL", "")),
		RPID:        getenv("WEBAUTHN_RP_ID", ""),
		DisplayName: getenv("WEBAUTHN_RP_DISPLAY_NAME", "Order Fill"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
