package domain

import (
	"errors"
	"testing"
	"time"
)

func TestValidateUploadsRejectsMoreThanTwoOrderFillBlanks(t *testing.T) {
	t.Parallel()
	err := ValidateUploads(TypeOrderFill, []UploadMeta{
		{Role: RoleSource, Name: "s.xlsx"},
		{Role: RoleBlank, Name: "a.xlsx"},
		{Role: RoleBlank, Name: "b.xlsx"},
		{Role: RoleBlank, Name: "c.xlsx"},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestNewJobRequiresOwner(t *testing.T) {
	t.Parallel()
	_, err := NewJob("j1", TypeOrderFill, "", "co", MatchingModeStandard, time.Now(), []FileRef{{ID: "f"}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestPurchaserSeesOwnJobsOnly(t *testing.T) {
	t.Parallel()
	job := Job{ID: "j1", CompanyID: "co", OwnerUserID: "u1"}
	buyer := Actor{UserID: "u1", CompanyID: "co", Role: RolePurchaser}
	other := Actor{UserID: "u2", CompanyID: "co", Role: RolePurchaser}
	admin := Actor{UserID: "a1", CompanyID: "co", Role: RoleCompanyAdmin}
	if !CanAccessJob(buyer, job) || CanAccessJob(other, job) || !CanAccessJob(admin, job) {
		t.Fatal("authz mismatch")
	}
}

func TestCompletedJobAcceptsEdits(t *testing.T) {
	t.Parallel()
	job := Job{Status: StatusCompleted}
	if !job.CanAcceptEdits() {
		t.Fatal("completed must accept edits")
	}
	job.Status = StatusProcessing
	if job.CanAcceptEdits() {
		t.Fatal("processing must not accept edits")
	}
}
