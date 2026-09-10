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
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
	DatabaseURL string
}

func Load() Config {
	env := getenv("FILE_ENV", getenv("APP_ENV", "local"))
	return Config{
		GRPCAddr:    getenv("FILE_GRPC_ADDR", ":9095"),
		HealthAddr:  getenv("FILE_HEALTH_ADDR", ":8086"),
		Environment: env,
		S3Endpoint:  getenv("FILE_S3_ENDPOINT", ""),
		S3AccessKey: getenv("FILE_S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey: getenv("FILE_S3_SECRET_KEY", "minioadmin"),
		S3Bucket:    getenv("FILE_S3_BUCKET", "order-fill"),
		S3UseSSL:    s3UseSSL(env),
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
	if strings.TrimSpace(c.S3Endpoint) == "" {
		return fmt.Errorf("FILE_S3_ENDPOINT is required outside local environment")
	}
	if !c.S3UseSSL {
		return fmt.Errorf("FILE_S3_USE_SSL must be true outside local environment")
	}
	if strings.TrimSpace(c.S3AccessKey) == "" || strings.TrimSpace(c.S3SecretKey) == "" {
		return fmt.Errorf("FILE_S3_ACCESS_KEY and FILE_S3_SECRET_KEY are required outside local environment")
	}
	if c.S3AccessKey == "minioadmin" || c.S3SecretKey == "minioadmin" {
		return fmt.Errorf("default MinIO credentials are not allowed outside local environment")
	}
	return nil
}

func s3UseSSL(env string) bool {
	if raw := os.Getenv("FILE_S3_USE_SSL"); raw != "" {
		return raw == "true"
	}
	return !localEnv(env)
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
