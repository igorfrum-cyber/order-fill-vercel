package migrate

import (
	"strings"
	"testing"
)

func TestBrandPolicyMigrationOwnsOverrideTable(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00001_brand_policy_overrides.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, expected := range []string{"brand_policy_overrides", "policy JSONB", "updated_by", "updated_at"} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
}
