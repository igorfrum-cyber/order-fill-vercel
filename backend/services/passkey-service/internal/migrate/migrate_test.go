package migrate

import (
	"strings"
	"testing"
)

func TestInitSQLAddsWebAuthnColumnsOnLegacyCredentials(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, needle := range []string{
		"ADD COLUMN IF NOT EXISTS public_key",
		"ADD COLUMN IF NOT EXISTS raw",
		"ADD COLUMN IF NOT EXISTS sign_count",
		"ADD COLUMN IF NOT EXISTS transports",
	} {
		if !strings.Contains(sql, needle) {
			t.Fatalf("legacy passkey_credentials needs %q", needle)
		}
	}
}
