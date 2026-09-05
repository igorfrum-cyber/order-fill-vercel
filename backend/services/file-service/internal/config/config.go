package config

import "os"

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
	return Config{
		GRPCAddr:    getenv("FILE_GRPC_ADDR", ":9095"),
		HealthAddr:  getenv("FILE_HEALTH_ADDR", ":8086"),
		Environment: getenv("FILE_ENV", "local"),
		S3Endpoint:  getenv("FILE_S3_ENDPOINT", ""),
		S3AccessKey: getenv("FILE_S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey: getenv("FILE_S3_SECRET_KEY", "minioadmin"),
		S3Bucket:    getenv("FILE_S3_BUCKET", "order-fill"),
		S3UseSSL:    getenv("FILE_S3_USE_SSL", "") == "true",
		DatabaseURL: getenv("DATABASE_URL", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
