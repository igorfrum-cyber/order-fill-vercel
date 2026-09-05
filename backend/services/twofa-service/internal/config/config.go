package config

import "os"

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
	MasterKey   string
	DatabaseURL string
	RedisURL    string
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("TWOFA_GRPC_ADDR", ":9092"),
		HealthAddr:  getenv("TWOFA_HEALTH_ADDR", ":8083"),
		Environment: getenv("TWOFA_ENV", "local"),
		MasterKey:   getenv("TWOFA_MASTER_KEY", "local-dev-twofa-master-key"),
		DatabaseURL: getenv("DATABASE_URL", ""),
		RedisURL:    getenv("QUEUE_URL", getenv("REDIS_URL", "")),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
