package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
	DatabaseURL string
}

func Load() Config {
	env := getenv("AUDIT_ENV", getenv("APP_ENV", "local"))
	return Config{
		GRPCAddr:    getenv("AUDIT_GRPC_ADDR", ":9100"),
		HealthAddr:  getenv("AUDIT_HEALTH_ADDR", ":8091"),
		Environment: env,
		DatabaseURL: getenv("DATABASE_URL", ""),
	}
}

func (c Config) Validate() error {
	if localEnv(c.Environment) {
		return nil
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required outside local environment")
	}
	return nil
}

func localEnv(env string) bool {
	return strings.EqualFold(strings.TrimSpace(env), "local") || strings.TrimSpace(env) == ""
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
