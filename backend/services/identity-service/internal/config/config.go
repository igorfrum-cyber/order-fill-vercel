package config

import (
	"fmt"
	"os"
	"strings"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/securecfg"
)

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
	env := getenv("IDENTITY_ENV", getenv("APP_ENV", "local"))
	return Config{
		GRPCAddr:            getenv("IDENTITY_GRPC_ADDR", ":9091"),
		HealthAddr:          getenv("IDENTITY_HEALTH_ADDR", ":8082"),
		Environment:         env,
		BootstrapAdminLogin: getenv("BOOTSTRAP_ADMIN_LOGIN", "admin"),
		TwoFAAddr:           getenv("TWOFA_GRPC_ADDR", ""),
		PasskeyAddr:         getenv("PASSKEY_GRPC_ADDR", ""),
		DatabaseURL:         getenv("DATABASE_URL", ""),
	}
}

func (c Config) Validate() error {
	if localEnv(c.Environment) {
		return nil
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required outside local environment")
	}
	if strings.TrimSpace(c.TwoFAAddr) == "" {
		return fmt.Errorf("TWOFA_GRPC_ADDR is required outside local environment")
	}
	if strings.TrimSpace(c.PasskeyAddr) == "" {
		return fmt.Errorf("PASSKEY_GRPC_ADDR is required outside local environment")
	}
	if err := securecfg.Postgres(c.Environment, c.DatabaseURL); err != nil {
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
