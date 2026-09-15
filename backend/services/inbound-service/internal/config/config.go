package config

import (
	"fmt"
	"os"
	"strings"

	"order-fill/backend/pkg/grpcutil"
)

type Config struct {
	GRPCAddr    string
	HealthAddr  string
	Environment string
	DatabaseURL string
	WorkerToken string
	InboundS3   S3Config
}

type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func Load() Config {
	return Config{
		GRPCAddr:    getenv("INBOUND_GRPC_ADDR", ":9101"),
		HealthAddr:  getenv("INBOUND_HEALTH_ADDR", ":8093"),
		Environment: getenv("INBOUND_ENV", getenv("APP_ENV", "local")),
		DatabaseURL: getenv("INBOUND_DATABASE_URL", ""),
		WorkerToken: getenv("WORKER_TOKEN", "local-dev-worker-token"),
		InboundS3: S3Config{
			Endpoint:  getenv("INBOUND_S3_ENDPOINT", "minio:9000"),
			AccessKey: getenv("INBOUND_S3_ACCESS_KEY", "minioadmin"),
			SecretKey: getenv("INBOUND_S3_SECRET_KEY", "minioadmin"),
			Bucket:    getenv("INBOUND_S3_BUCKET", "order-fill-inbound"),
			UseSSL:    strings.EqualFold(getenv("INBOUND_S3_USE_SSL", ""), "true"),
		},
	}
}

func (c Config) Validate() error {
	if !localEnv(c.Environment) {
		if strings.TrimSpace(c.DatabaseURL) == "" {
			return fmt.Errorf("INBOUND_DATABASE_URL is required outside local environment")
		}
	}
	return grpcutil.CheckWorkerToken(c.Environment, c.WorkerToken)
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
