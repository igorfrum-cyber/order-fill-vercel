package securecfg

import (
	"fmt"
	"net/url"
	"strings"
)

const defaultPostgresPassword = "order_fill"

func Local(env string) bool {
	return strings.EqualFold(strings.TrimSpace(env), "local") || strings.TrimSpace(env) == ""
}

func Postgres(env, databaseURL string) error {
	if Local(env) || strings.TrimSpace(databaseURL) == "" {
		return nil
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}
	if parsed.User != nil {
		if password, ok := parsed.User.Password(); ok && password == defaultPostgresPassword {
			return fmt.Errorf("DATABASE_URL must not use the default password outside local environment")
		}
	}
	sslmode := strings.ToLower(parsed.Query().Get("sslmode"))
	switch sslmode {
	case "require", "verify-ca", "verify-full":
		return nil
	default:
		return fmt.Errorf("DATABASE_URL must set sslmode=require outside local environment")
	}
}

func Redis(env, redisURL string) error {
	if Local(env) || strings.TrimSpace(redisURL) == "" {
		return nil
	}
	parsed, err := url.Parse(redisURL)
	if err != nil {
		return fmt.Errorf("Redis URL is invalid: %w", err)
	}
	if parsed.User == nil {
		return fmt.Errorf("Redis URL must include a password outside local environment")
	}
	password, ok := parsed.User.Password()
	if !ok || password == "" {
		return fmt.Errorf("Redis URL must include a password outside local environment")
	}
	return nil
}
