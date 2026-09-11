package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/securecfg"
)

type Config struct {
	GRPCAddr     string
	HealthAddr   string
	Environment  string
	S3Endpoint   string
	S3AccessKey  string
	S3SecretKey  string
	S3Bucket     string
	S3UseSSL     bool
	DatabaseURL  string
	IdentityAddr string
	WorkerToken  string
}

func Load() Config {
	env := getenv("FILE_ENV", getenv("APP_ENV", "local"))
	return Config{
		GRPCAddr:     getenv("FILE_GRPC_ADDR", ":9095"),
		HealthAddr:   getenv("FILE_HEALTH_ADDR", ":8086"),
		Environment:  env,
		S3Endpoint:   getenv("FILE_S3_ENDPOINT", ""),
		S3AccessKey:  getenv("FILE_S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:  getenv("FILE_S3_SECRET_KEY", "minioadmin"),
		S3Bucket:     getenv("FILE_S3_BUCKET", "order-fill"),
		S3UseSSL:     s3UseSSL(env),
		DatabaseURL:  getenv("DATABASE_URL", ""),
		IdentityAddr: getenv("IDENTITY_GRPC_ADDR", ""),
		WorkerToken:  getenv("WORKER_TOKEN", ""),
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
	if err := rejectHTTPEndpoint(c.S3Endpoint); err != nil {
		return err
	}
	if err := securecfg.Postgres(c.Environment, c.DatabaseURL); err != nil {
		return err
	}
	if strings.TrimSpace(c.IdentityAddr) == "" {
		return fmt.Errorf("IDENTITY_GRPC_ADDR is required outside local environment")
	}
	if err := grpcutil.CheckWorkerToken(c.Environment, c.WorkerToken); err != nil {
		return err
	}
	return grpcutil.CheckTLSMode(c.Environment)
}

func s3UseSSL(env string) bool {
	if raw := os.Getenv("FILE_S3_USE_SSL"); raw != "" {
		return raw == "true"
	}
	return !localEnv(env)
}

func rejectHTTPEndpoint(endpoint string) error {
	trimmed := strings.TrimSpace(endpoint)
	if !strings.Contains(trimmed, "://") {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("FILE_S3_ENDPOINT is invalid: %w", err)
	}
	if strings.EqualFold(parsed.Scheme, "http") {
		return fmt.Errorf("FILE_S3_ENDPOINT must not use http outside local environment")
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
