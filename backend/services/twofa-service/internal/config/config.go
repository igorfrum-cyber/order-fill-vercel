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
	MasterKey   string
	DatabaseURL string
	RedisURL    string
}

func Load() Config {
	env := getenv("TWOFA_ENV", getenv("APP_ENV", "local"))
	return Config{
		GRPCAddr:    getenv("TWOFA_GRPC_ADDR", ":9092"),
		HealthAddr:  getenv("TWOFA_HEALTH_ADDR", ":8083"),
		Environment: env,
		MasterKey:   getenv("TWOFA_MASTER_KEY", "local-dev-twofa-master-key"),
		DatabaseURL: getenv("DATABASE_URL", ""),
		RedisURL:    getenv("QUEUE_URL", getenv("REDIS_URL", "")),
	}
}

func (c Config) Validate() error {
	if localEnv(c.Environment) {
		return nil
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required outside local environment")
	}
	if strings.TrimSpace(c.RedisURL) == "" {
		return fmt.Errorf("QUEUE_URL or REDIS_URL is required outside local environment")
	}
	if c.MasterKey == "" || c.MasterKey == "local-dev-twofa-master-key" || len(c.MasterKey) < 32 {
		return fmt.Errorf("TWOFA_MASTER_KEY must be a non-default value with at least 32 bytes outside local environment")
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
