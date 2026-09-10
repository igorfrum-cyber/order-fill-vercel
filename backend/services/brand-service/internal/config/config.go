package config

import "os"

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("BRAND_GRPC_ADDR", ":9098"),
		HealthAddr:  getenv("BRAND_HEALTH_ADDR", ":8089"),
		Environment: getenv("BRAND_ENV", "local"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
