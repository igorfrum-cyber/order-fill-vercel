package config

import "os"

type Config struct {
	GRPCAddr        string
	HealthAddr      string
	Environment     string
	QueueURL        string
	JobAddr         string
	FileAddr        string
	CalculationAddr string
}

func Load() Config {
	return Config{
		GRPCAddr:        getenv("DOCUMENT_GRPC_ADDR", ":9096"),
		HealthAddr:      getenv("DOCUMENT_HEALTH_ADDR", ":8087"),
		Environment:     getenv("DOCUMENT_ENV", "local"),
		QueueURL:        getenv("QUEUE_URL", ""),
		JobAddr:         getenv("JOB_GRPC_ADDR", ""),
		FileAddr:        getenv("FILE_GRPC_ADDR", ""),
		CalculationAddr: getenv("CALCULATION_GRPC_ADDR", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
