package config

import (
	"fmt"
	"os"
	"strings"
)

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
	env := getenv("JOB_ENV", getenv("APP_ENV", "local"))
	return Config{
		GRPCAddr:     getenv("JOB_GRPC_ADDR", ":9094"),
		HealthAddr:   getenv("JOB_HEALTH_ADDR", ":8085"),
		Environment:  env,
		QueueURL:     getenv("QUEUE_URL", ""),
		FileAddr:     getenv("FILE_GRPC_ADDR", ""),
		IdentityAddr: getenv("IDENTITY_GRPC_ADDR", ""),
		DatabaseURL:  getenv("DATABASE_URL", ""),
	}
}

func (c Config) Validate() error {
	if localEnv(c.Environment) {
		return nil
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required outside local environment")
	}
	if strings.TrimSpace(c.QueueURL) == "" {
		return fmt.Errorf("QUEUE_URL is required outside local environment")
	}
	if strings.TrimSpace(c.FileAddr) == "" {
		return fmt.Errorf("FILE_GRPC_ADDR is required outside local environment")
	}
	if strings.TrimSpace(c.IdentityAddr) == "" {
		return fmt.Errorf("IDENTITY_GRPC_ADDR is required outside local environment")
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
