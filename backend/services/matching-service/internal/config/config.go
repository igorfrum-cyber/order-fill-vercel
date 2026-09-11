package config

import (
	"os"

	"order-fill/backend/pkg/grpcutil"
)

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("MATCHING_GRPC_ADDR", ":9097"),
		HealthAddr:  getenv("MATCHING_HEALTH_ADDR", ":8088"),
		Environment: getenv("MATCHING_ENV", getenv("APP_ENV", "local")),
	}
}

func (c Config) Validate() error {
	return grpcutil.CheckTLSMode(c.Environment)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
