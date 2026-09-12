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
	owner := Actor{UserID: "o1", CompanyID: "co", Role: RoleCompanyOwner}
	foreign := Actor{UserID: "a2", CompanyID: "other", Role: RoleCompanyAdmin}
	platform := Actor{UserID: "p1", Role: RolePlatformAdmin}
	if !CanAccessJob(buyer, job) || CanAccessJob(other, job) || !CanAccessJob(admin, job) {
		t.Fatal("authz mismatch")
	}
	if !CanAccessJob(owner, job) || CanAccessJob(foreign, job) || !CanAccessJob(platform, job) {
		t.Fatal("owner/platform/cross-company mismatch")
	}
}

func TestCanCreateJob(t *testing.T) {
	t.Parallel()
	if !CanCreateJob(Actor{Role: RolePurchaser, CompanyID: "co"}) {
		t.Fatal("purchaser")
	}
	if !CanCreateJob(Actor{Role: RoleCompanyOwner, CompanyID: "co"}) || !CanCreateJob(Actor{Role: RoleCompanyAdmin, CompanyID: "co"}) {
		t.Fatal("keepers")
	}
	if CanCreateJob(Actor{Role: RolePlatformAdmin}) || CanCreateJob(Actor{Role: RolePurchaser}) {
		t.Fatal("platform admin and purchaser without company")
	}
	if CanCreateJob(Actor{Role: RolePurchaser, CompanyID: "co", Disabled: true}) {
		t.Fatal("disabled")
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

func TestValidateUploadsWarehouseRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		jobType Type
		uploads []UploadMeta
		wantErr bool
	}{
		{
			name: "order fill accepts warehouse", jobType: TypeOrderFill,
			uploads: []UploadMeta{{Role: RoleSource, Name: "office.xlsx"}, {Role: RoleWarehouse, Name: "stock.xlsx"}, {Role: RoleBlank, Name: "blank.xlsx"}},
		},
		{
			name: "north accepts warehouse without source", jobType: TypeNorthMerge,
			uploads: []UploadMeta{{Role: RoleBlank, Name: "surgut.xlsx"}, {Role: RoleWarehouse, Name: "stock.xlsx"}},
		},
		{
			name: "duplicate name is rejected", jobType: TypeOrderFill, wantErr: true,
			uploads: []UploadMeta{{Role: RoleSource, Name: "same.xlsx"}, {Role: RoleWarehouse, Name: " SAME.xlsx "}, {Role: RoleBlank, Name: "blank.xlsx"}},
		},
		{
			name: "second warehouse is rejected", jobType: TypeNorthMerge, wantErr: true,
			uploads: []UploadMeta{{Role: RoleBlank, Name: "surgut.xlsx"}, {Role: RoleWarehouse, Name: "one.xlsx"}, {Role: RoleWarehouse, Name: "two.xlsx"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateUploads(tc.jobType, tc.uploads)
			if tc.wantErr && !errors.Is(err, ErrInvalid) {
				t.Fatalf("got %v, want invalid upload", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatal(err)
			}
		})
	}
}
