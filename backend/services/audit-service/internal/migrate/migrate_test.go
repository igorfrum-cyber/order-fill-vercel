package migrate

import (
	"strings"
	"testing"
)

func TestInitSQLUpgradesLegacyAuditEvents(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, needle := range []string{
		"ADD COLUMN IF NOT EXISTS created_at",
		"ADD COLUMN IF NOT EXISTS type",
		"ADD COLUMN IF NOT EXISTS payload",
	} {
		if !strings.Contains(sql, needle) {
			t.Fatalf("legacy audit_events needs %q", needle)
		}
	}
}
