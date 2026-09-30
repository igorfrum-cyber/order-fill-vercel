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

func TestNorthAcceptsChristinaVariantRoles(t *testing.T) {
	t.Parallel()
	err := ValidateUploads(TypeNorthMerge, []UploadMeta{
		{Role: RoleBlankHome, Name: "surgut-home.xlsx"},
		{Role: RoleBlankProff, Name: "surgut-proff.xlsx"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateUploads(TypeOrderFill, []UploadMeta{{Role: RoleSource, Name: "source.xlsx"}, {Role: RoleBlankHome, Name: "home.xlsx"}}); err == nil {
		t.Fatal("order fill must reject north-only variant role")
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
	if CanAccessJob(Actor{Role: RoleCompanyOwner, CompanyID: "co", Disabled: true}, job) ||
		CanAccessJob(Actor{Role: "unknown", CompanyID: "co"}, job) {
		t.Fatal("disabled or unknown actors must not access jobs")
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

func TestParseTypeAndMatchingMode(t *testing.T) {
	t.Parallel()

	for _, jobType := range []Type{TypeOrderFill, TypeNorthMerge} {
		got, err := ParseType(string(jobType))
		if err != nil || got != jobType {
			t.Fatalf("ParseType(%q) = %q, %v", jobType, got, err)
		}
	}
	if _, err := ParseType("unsupported"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ParseType unsupported error = %v", err)
	}
	if ParseMatchingMode("smart") != MatchingModeSmart || ParseMatchingMode("unknown") != MatchingModeStandard {
		t.Fatal("matching mode parsing failed")
	}
}

func TestNewJobValidatesFieldsAndSetsDefaults(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 2, 3, 4, 5, 0, time.FixedZone("test", 3600))
	files := []FileRef{{ID: "file-1"}}
	job, err := NewJob("job-1", TypeNorthMerge, "owner", "company", "", now, files)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != StatusQueued || job.MatchingMode != MatchingModeStandard || !job.CreatedAt.Equal(now.UTC()) || !job.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("new job = %+v", job)
	}
	if len(job.InputFiles) != 1 || job.OutputFiles == nil || len(job.OutputFiles) != 0 {
		t.Fatalf("new job files = %+v / %+v", job.InputFiles, job.OutputFiles)
	}

	tests := []struct {
		name    string
		id      string
		jobType Type
		owner   string
		company string
		files   []FileRef
	}{
		{name: "blank id", id: " ", jobType: TypeOrderFill, owner: "owner", company: "company", files: files},
		{name: "unknown type", id: "job", jobType: "unknown", owner: "owner", company: "company", files: files},
		{name: "blank owner", id: "job", jobType: TypeOrderFill, owner: " ", company: "company", files: files},
		{name: "blank company", id: "job", jobType: TypeOrderFill, owner: "owner", company: " ", files: files},
		{name: "no files", id: "job", jobType: TypeOrderFill, owner: "owner", company: "company"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewJob(tt.id, tt.jobType, tt.owner, tt.company, MatchingModeStandard, now, tt.files); !errors.Is(err, ErrInvalid) {
				t.Fatalf("NewJob() error = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestValidateUploadsRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	validSourceAndBlank := []UploadMeta{{Role: RoleSource, Name: "source.xlsx"}, {Role: RoleBlank, Name: "blank.xlsx"}}
	tests := []struct {
		name    string
		jobType Type
		uploads []UploadMeta
	}{
		{name: "no uploads", jobType: TypeOrderFill},
		{name: "blank file name", jobType: TypeNorthMerge, uploads: []UploadMeta{{Role: RoleBlank, Name: " "}}},
		{name: "non workbook extension", jobType: TypeNorthMerge, uploads: []UploadMeta{{Role: RoleBlank, Name: "blank.xls"}}},
		{name: "unknown role", jobType: TypeNorthMerge, uploads: []UploadMeta{{Role: "other", Name: "blank.xlsx"}}},
		{name: "no order fill blank", jobType: TypeOrderFill, uploads: []UploadMeta{{Role: RoleSource, Name: "source.xlsx"}}},
		{name: "no order fill source", jobType: TypeOrderFill, uploads: []UploadMeta{{Role: RoleBlank, Name: "blank.xlsx"}}},
		{name: "multiple order fill sources", jobType: TypeOrderFill, uploads: append(validSourceAndBlank, UploadMeta{Role: RoleSource, Name: "source2.xlsm"})},
		{name: "variant blank is North only", jobType: TypeOrderFill, uploads: []UploadMeta{{Role: RoleSource, Name: "source.xlsx"}, {Role: RoleBlankHome, Name: "home.xlsx"}}},
		{name: "unsupported type", jobType: "unknown", uploads: []UploadMeta{{Role: RoleBlank, Name: "blank.xlsx"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateUploads(tt.jobType, tt.uploads); !errors.Is(err, ErrInvalid) {
				t.Fatalf("ValidateUploads() error = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestValidateUploadsAcceptsWorkbookExtensionsAndNorthWithoutSource(t *testing.T) {
	t.Parallel()

	if err := ValidateUploads(TypeOrderFill, []UploadMeta{
		{Role: RoleSource, Name: "source.XLSM"}, {Role: RoleBlank, Name: "blank.XLSX"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUploads(TypeNorthMerge, []UploadMeta{{Role: RoleBlank, Name: "blank.xlsx"}}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUploads(TypeNorthMerge, []UploadMeta{{Role: RoleSource, Name: "source.xlsx"}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("North merge without a blank error = %v", err)
	}
}

func TestSafeFileNameAndFail(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ input, want string }{
		{" ../../orders.xlsx ", "orders.xlsx"},
		{`C:\\shared\\orders.xlsx`, "orders.xlsx"},
		{"/", "file"},
		{"", "file"},
	} {
		if got := SafeFileName(tc.input); got != tc.want {
			t.Errorf("SafeFileName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}

	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.FixedZone("test", -7200))
	job := Job{}
	job.Fail("invalid workbook", now)
	if job.Status != StatusFailed || job.ErrorMessage != "invalid workbook" || !job.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("failed job = %+v", job)
	}
}
