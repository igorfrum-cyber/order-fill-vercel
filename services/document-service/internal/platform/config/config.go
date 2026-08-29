// Package config reads the worker environment once, at startup.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HealthAddr  string
	DatabaseURL string
	QueueURL    string
	QueueName   string
	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
}

// Load reads the configuration and fails fast when a required value is missing.
func Load() (Config, error) {
	config := Config{
		HealthAddr:  getenv("WORKER_HEALTH_ADDR", ":8081"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		QueueURL:    os.Getenv("QUEUE_URL"),
		QueueName:   getenv("QUEUE_NAME", "order-fill:jobs"),
		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3Bucket:    getenv("S3_BUCKET", "order-fill"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
	}

	missing := make([]string, 0)
	for name, value := range map[string]string{
		"DATABASE_URL":  config.DatabaseURL,
		"QUEUE_URL":     config.QueueURL,
		"S3_ENDPOINT":   config.S3Endpoint,
		"S3_ACCESS_KEY": config.S3AccessKey,
		"S3_SECRET_KEY": config.S3SecretKey,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return config, nil
}

func getenv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
