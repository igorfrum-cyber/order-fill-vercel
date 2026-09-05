package config

import "os"

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
	DatabaseURL string
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("AUDIT_GRPC_ADDR", ":9100"),
		HealthAddr:  getenv("AUDIT_HEALTH_ADDR", ":8091"),
		Environment: getenv("AUDIT_ENV", "local"),
		DatabaseURL: getenv("DATABASE_URL", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
