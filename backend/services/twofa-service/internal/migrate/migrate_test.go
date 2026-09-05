package migrate

import (
	"strings"
	"testing"
)

func TestInitSQLAddsEnabledOnLegacyTotp(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "ADD COLUMN IF NOT EXISTS enabled") {
		t.Fatal("legacy user_totp needs enabled")
	}
}
