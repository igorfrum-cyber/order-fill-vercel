package usecase

import (
	"context"
	"testing"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func TestCreateCompanyAssignsLoginSlug(t *testing.T) {
	store := newMemoryIdentity()
	admin := NewAdmin(store, func() string { return "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	actor := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	company, err := admin.CreateCompany(context.Background(), actor, "Acme")
	if err != nil {
		t.Fatal(err)
	}
	if company.LoginSlug != "acme" {
		t.Fatalf("login slug: got %q", company.LoginSlug)
	}
}

func TestCreateCompanyMakesDuplicateNamesUnique(t *testing.T) {
	store := newMemoryIdentity()
	n := 0
	admin := NewAdmin(store, func() string {
		n++
		if n == 1 {
			return "11111111-1111-1111-1111-111111111111"
		}
		return "22222222-2222-2222-2222-222222222222"
	}, func() time.Time { return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC) })
	actor := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if _, err := admin.CreateCompany(context.Background(), actor, "Acme"); err != nil {
		t.Fatal(err)
	}
	second, err := admin.CreateCompany(context.Background(), actor, "Acme")
	if err != nil {
		t.Fatal(err)
	}
	if second.LoginSlug != "acme-22222222" {
		t.Fatalf("login slug: got %q", second.LoginSlug)
	}
}

func TestPublicCompanyLoginReturnsActiveCompany(t *testing.T) {
	store := newMemoryIdentity()
	admin := NewAdmin(store, func() string { return "id-1" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	if err := store.CreateCompany(context.Background(), identity.Company{ID: "co-1", Name: "Acme", LoginSlug: "acme"}); err != nil {
		t.Fatal(err)
	}
	company, err := admin.PublicCompanyLogin(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if company.Name != "Acme" || company.LoginSlug != "acme" {
		t.Fatalf("got %+v", company)
	}
}

func TestPublicCompanyLoginHidesDisabledAndUnknown(t *testing.T) {
	store := newMemoryIdentity()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	admin := NewAdmin(store, func() string { return "id-1" }, func() time.Time { return now })
	if err := store.CreateCompany(context.Background(), identity.Company{ID: "co-1", Name: "Gone", LoginSlug: "gone"}); err != nil {
		t.Fatal(err)
	}
	if err := store.DisableCompany(context.Background(), "co-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.PublicCompanyLogin(context.Background(), "gone"); err != identity.ErrNotFound {
		t.Fatalf("disabled: %v", err)
	}
	if _, err := admin.PublicCompanyLogin(context.Background(), "missing"); err != identity.ErrNotFound {
		t.Fatalf("unknown: %v", err)
	}
}

func TestListJobsScopesByRole(t *testing.T) {
	repo := &listJobRepo{}
	lister := NewListJobs(repo)
	ctx := context.Background()

	purchaser := identity.User{ID: "u1", CompanyID: "c1", Role: identity.RolePurchaser}
	if _, err := lister.Execute(ctx, purchaser, "ignored"); err != nil {
		t.Fatal(err)
	}
	if repo.last.CompanyID != "c1" || repo.last.CreatedBy != "u1" {
		t.Fatalf("purchaser filter %+v", repo.last)
	}

	admin := identity.User{ID: "a1", CompanyID: "c1", Role: identity.RoleCompanyAdmin}
	if _, err := lister.Execute(ctx, admin, "other"); err != nil {
		t.Fatal(err)
	}
	if repo.last.CompanyID != "c1" || repo.last.CreatedBy != "" {
		t.Fatalf("company admin filter %+v", repo.last)
	}

	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if _, err := lister.Execute(ctx, platform, "c9"); err != nil {
		t.Fatal(err)
	}
	if repo.last.CompanyID != "c9" || repo.last.CreatedBy != "" {
		t.Fatalf("platform filter %+v", repo.last)
	}
}

type listJobRepo struct {
	last port.JobListFilter
}

func (r *listJobRepo) List(_ context.Context, filter port.JobListFilter) ([]port.JobListRow, error) {
	r.last = filter
	return []port.JobListRow{{Job: job.Job{ID: "job-1"}}}, nil
}
