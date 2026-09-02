package postgres

import (
	"strings"
	"testing"
)

func TestMigrateStatementsIncludeUsers(t *testing.T) {
	joined := strings.Join(migrateStatements(), "\n")
	for _, needle := range []string{
		"CREATE TABLE IF NOT EXISTS users",
		"CREATE TABLE IF NOT EXISTS sessions",
		"CREATE TABLE IF NOT EXISTS invite_tokens",
		"CREATE TABLE IF NOT EXISTS companies",
		"created_by",
		"login_slug",
		"companies_login_slug_uidx",
		"logo_content_type",
		"CREATE TABLE IF NOT EXISTS user_totp",
		"recovery_code_hashes JSONB NOT NULL DEFAULT '[]'",
	} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("missing %s", needle)
		}
	}
}
