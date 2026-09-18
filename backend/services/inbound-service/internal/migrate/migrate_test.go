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
