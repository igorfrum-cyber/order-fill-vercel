package migrate

import (
	"io/fs"
	"strings"
	"testing"
)

func TestMigrationPrefixesAreUnique(t *testing.T) {
	t.Parallel()
	entries, err := fs.ReadDir(files, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, e := range entries {
		prefix := strings.SplitN(e.Name(), "_", 2)[0]
		if other, ok := seen[prefix]; ok {
			t.Fatalf("duplicate migration prefix %s: %s and %s", prefix, other, e.Name())
		}
		seen[prefix] = e.Name()
	}
}

func TestSenderEmailUniqueIndexMigration(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00006_unique_sender_email.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, needle := range []string{
		"company_inbound_sender_email_lower_uidx",
		"LOWER(sender_email)",
		"WHERE sender_email <> ''",
	} {
		if !strings.Contains(sql, needle) {
			t.Fatalf("unique sender migration needs %q", needle)
		}
	}
}
