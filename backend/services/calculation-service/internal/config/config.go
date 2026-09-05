package config

import "os"

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("CALCULATION_GRPC_ADDR", ":9099"),
		HealthAddr:  getenv("CALCULATION_HEALTH_ADDR", ":8090"),
		Environment: getenv("CALCULATION_ENV", "local"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
