package migrate

import (
	"strings"
	"testing"
)

func TestInitSQLAddsCompanyColumnsOnLegacyTable(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, needle := range []string{
		"ADD COLUMN IF NOT EXISTS matching_mode",
		"ADD COLUMN IF NOT EXISTS login_slug",
		"ADD COLUMN IF NOT EXISTS logo_content_type",
	} {
		if !strings.Contains(sql, needle) {
			t.Fatalf("legacy companies needs %q", needle)
		}
	}
}
