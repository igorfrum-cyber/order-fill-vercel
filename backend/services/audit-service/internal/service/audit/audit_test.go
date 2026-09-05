package audit_test

import (
	"testing"
	"time"

	"order-fill/backend/services/audit-service/internal/service/audit"
	"order-fill/backend/services/audit-service/internal/storage/memory"
)

func TestRecordAndListByCompany(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	svc := audit.New(memory.New(), func() time.Time { return now })
	id, err := svc.Record("login_success", "u1", "co-1", "", "")
	if err != nil || id == "" {
		t.Fatal(err)
	}
	if _, err := svc.Record("job_view", "u2", "co-2", "j1", "{}"); err != nil {
		t.Fatal(err)
	}
	got := svc.List("co-1")
	if len(got) != 1 || got[0].Type != "login_success" {
		t.Fatalf("%+v", got)
	}
}
