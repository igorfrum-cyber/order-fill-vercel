package config

import "os"

type Config struct {
	GRPCAddr            string
	HealthAddr          string
	Environment         string
	BootstrapAdminLogin string
	TwoFAAddr           string
	PasskeyAddr         string
	DatabaseURL         string
}

func Load() Config {
	return Config{
		GRPCAddr:            getenv("IDENTITY_GRPC_ADDR", ":9091"),
		HealthAddr:          getenv("IDENTITY_HEALTH_ADDR", ":8082"),
		Environment:         getenv("IDENTITY_ENV", "local"),
		BootstrapAdminLogin: getenv("BOOTSTRAP_ADMIN_LOGIN", "admin"),
		TwoFAAddr:           getenv("TWOFA_GRPC_ADDR", ""),
		PasskeyAddr:         getenv("PASSKEY_GRPC_ADDR", ""),
		DatabaseURL:         getenv("DATABASE_URL", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
