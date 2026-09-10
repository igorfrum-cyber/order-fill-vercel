package config

import "os"

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("MATCHING_GRPC_ADDR", ":9097"),
		HealthAddr:  getenv("MATCHING_HEALTH_ADDR", ":8088"),
		Environment: getenv("MATCHING_ENV", "local"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
