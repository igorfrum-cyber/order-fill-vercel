package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func TestCreateCompanyRequiresLoginSlug(t *testing.T) {
	admin, actor := newPlatformAdmin(t)
	if _, err := admin.CreateCompany(context.Background(), actor, "Кристайл", ""); !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateCompanyUsesProvidedLoginSlug(t *testing.T) {
	admin, actor := newPlatformAdmin(t)
	company, err := admin.CreateCompany(context.Background(), actor, "Кристайл", "Kristail")
	if err != nil {
		t.Fatal(err)
	}
	if company.LoginSlug != "kristail" {
		t.Fatalf("login slug: got %q", company.LoginSlug)
	}
}

func TestCreateCompanyRejectsDuplicateLoginSlug(t *testing.T) {
	admin, actor := newPlatformAdmin(t)
	if _, err := admin.CreateCompany(context.Background(), actor, "Acme", "acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.CreateCompany(context.Background(), actor, "Other", "acme"); !errors.Is(err, identity.ErrConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestSetCompanyLoginSlugUpdatesExisting(t *testing.T) {
	admin, actor := newPlatformAdmin(t)
	created, err := admin.CreateCompany(context.Background(), actor, "Кристайл", "oldslug")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := admin.SetCompanyLoginSlug(context.Background(), actor, created.ID, "kristail")
	if err != nil {
		t.Fatal(err)
	}
	if updated.LoginSlug != "kristail" {
		t.Fatalf("login slug: got %q", updated.LoginSlug)
	}
}

func newPlatformAdmin(t *testing.T) (*Admin, identity.User) {
	t.Helper()
	n := 0
	store := newMemoryIdentity()
	admin := NewAdmin(store, func() string {
		n++
		return fmt.Sprintf("co-%d", n)
	}, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	return admin, identity.User{ID: "root", Role: identity.RolePlatformAdmin}
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
