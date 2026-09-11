package config

import (
	"fmt"
	"os"
	"strings"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/securecfg"
)

type Config struct {
	GRPCAddr        string
	HealthAddr      string
	Environment     string
	QueueURL        string
	JobAddr         string
	FileAddr        string
	CalculationAddr string
	MatchingAddr    string
	BrandAddr       string
}

func Load() Config {
	env := getenv("DOCUMENT_ENV", getenv("APP_ENV", "local"))
	return Config{
		GRPCAddr:        getenv("DOCUMENT_GRPC_ADDR", ":9096"),
		HealthAddr:      getenv("DOCUMENT_HEALTH_ADDR", ":8087"),
		Environment:     env,
		QueueURL:        getenv("QUEUE_URL", ""),
		JobAddr:         getenv("JOB_GRPC_ADDR", ""),
		FileAddr:        getenv("FILE_GRPC_ADDR", ""),
		CalculationAddr: getenv("CALCULATION_GRPC_ADDR", ""),
		MatchingAddr:    getenv("MATCHING_GRPC_ADDR", ""),
		BrandAddr:       getenv("BRAND_GRPC_ADDR", ""),
	}
}

func (c Config) ValidateAPI() error {
	if localEnv(c.Environment) {
		return nil
	}
	if strings.TrimSpace(c.FileAddr) == "" {
		return fmt.Errorf("FILE_GRPC_ADDR is required outside local environment")
	}
	if strings.TrimSpace(c.BrandAddr) == "" {
		return fmt.Errorf("BRAND_GRPC_ADDR is required outside local environment")
	}
	return grpcutil.CheckTLSMode(c.Environment)
}

func (c Config) ValidateWorker() error {
	if localEnv(c.Environment) {
		return nil
	}
	required := map[string]string{
		"QUEUE_URL":             c.QueueURL,
		"JOB_GRPC_ADDR":         c.JobAddr,
		"FILE_GRPC_ADDR":        c.FileAddr,
		"CALCULATION_GRPC_ADDR": c.CalculationAddr,
		"MATCHING_GRPC_ADDR":    c.MatchingAddr,
		"BRAND_GRPC_ADDR":       c.BrandAddr,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required outside local environment", name)
		}
	}
	if err := securecfg.Redis(c.Environment, c.QueueURL); err != nil {
		return err
	}
	return grpcutil.CheckTLSMode(c.Environment)
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
