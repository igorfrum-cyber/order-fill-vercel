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

func TestChristinaProffModeMigrationAddsCompanyColumn(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00005_christina_proff_mode.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	if !strings.Contains(sql, "ADD COLUMN IF NOT EXISTS christina_proff_mode") {
		t.Fatal("companies needs christina_proff_mode")
	}
}

func TestPrimaryAdminMigrationProtectsExactlyOneBootstrapAdmin(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00004_primary_platform_admin.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, needle := range []string{"is_primary_admin", "role = 'platform_admin'", "CREATE UNIQUE INDEX"} {
		if !strings.Contains(sql, needle) {
			t.Fatalf("primary admin migration needs %q", needle)
		}
	}
}
