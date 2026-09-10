package migrate

import (
	"strings"
	"testing"
)

func TestInitSQLAddsOwnerAndModeOnLegacyJobs(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, needle := range []string{
		"ADD COLUMN IF NOT EXISTS owner_user_id",
		"ADD COLUMN IF NOT EXISTS matching_mode",
	} {
		if !strings.Contains(sql, needle) {
			t.Fatalf("legacy jobs needs %q", needle)
		}
	}
}

func TestJobFieldsSQLAddsReportPayload(t *testing.T) {
	t.Parallel()
	body, err := files.ReadFile("migrations/00002_job_fields.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "ADD COLUMN IF NOT EXISTS payload") {
		t.Fatal("legacy job_reports needs payload")
	}
}
