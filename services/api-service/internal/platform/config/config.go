// Package config reads the process environment once, at startup, so no other
// package has to know that configuration comes from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultMaxUploadBytes int64 = 64 << 20

type Config struct {
	Addr           string
	AllowedOrigins string
	DatabaseURL    string
	QueueURL       string
	QueueName      string
	S3Endpoint     string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	MaxUploadBytes int64
}

// Load reads the configuration and fails fast when a required value is missing,
// because a half-configured service is worse than one that refuses to start.
func Load() (Config, error) {
	config := Config{
		Addr:           getenv("API_ADDR", ":8080"),
		AllowedOrigins: getenv("API_ALLOWED_ORIGINS", "*"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		QueueURL:       os.Getenv("QUEUE_URL"),
		QueueName:      getenv("QUEUE_NAME", "order-fill:jobs"),
		S3Endpoint:     os.Getenv("S3_ENDPOINT"),
		S3Bucket:       getenv("S3_BUCKET", "order-fill"),
		S3AccessKey:    os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:    os.Getenv("S3_SECRET_KEY"),
		MaxUploadBytes: getenvInt64("API_MAX_UPLOAD_BYTES", defaultMaxUploadBytes),
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

func getenvInt64(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(key)), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
