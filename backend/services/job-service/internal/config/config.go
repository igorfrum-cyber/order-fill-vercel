package config

import "os"

type Config struct {
	GRPCAddr     string
	HealthAddr   string
	Environment  string
	QueueURL     string
	FileAddr     string
	IdentityAddr string
	DatabaseURL  string
}

func Load() Config {
	return Config{
		GRPCAddr:     getenv("JOB_GRPC_ADDR", ":9094"),
		HealthAddr:   getenv("JOB_HEALTH_ADDR", ":8085"),
		Environment:  getenv("JOB_ENV", "local"),
		QueueURL:     getenv("QUEUE_URL", ""),
		FileAddr:     getenv("FILE_GRPC_ADDR", ""),
		IdentityAddr: getenv("IDENTITY_GRPC_ADDR", ""),
		DatabaseURL:  getenv("DATABASE_URL", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
