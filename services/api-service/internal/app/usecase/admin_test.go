package usecase

import (
	"bytes"
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

func TestSetCompanyLoginSlugAllowsCompanyAdminForOwnCompany(t *testing.T) {
	admin, platform := newPlatformAdmin(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Кристайл", "oldslug")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "admin-a", Role: identity.RoleCompanyAdmin, CompanyID: created.ID}
	updated, err := admin.SetCompanyLoginSlug(context.Background(), actor, created.ID, "kristail")
	if err != nil {
		t.Fatal(err)
	}
	if updated.LoginSlug != "kristail" {
		t.Fatalf("login slug: got %q", updated.LoginSlug)
	}
}

func TestSetCompanyLoginSlugRejectsCompanyAdminForOtherCompany(t *testing.T) {
	admin, platform := newPlatformAdmin(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Кристайл", "oldslug")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "admin-b", Role: identity.RoleCompanyAdmin, CompanyID: "other"}
	if _, err := admin.SetCompanyLoginSlug(context.Background(), actor, created.ID, "kristail"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestSetCompanyLoginSlugRejectsPurchaser(t *testing.T) {
	admin, platform := newPlatformAdmin(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Кристайл", "oldslug")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "buyer", Role: identity.RolePurchaser, CompanyID: created.ID}
	if _, err := admin.SetCompanyLoginSlug(context.Background(), actor, created.ID, "kristail"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateCompanyAllowsCompanyAdminToEditOwnProfile(t *testing.T) {
	admin, platform := newPlatformAdmin(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Кристайл", "oldslug")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "admin-a", Role: identity.RoleCompanyAdmin, CompanyID: created.ID}
	updated, err := admin.UpdateCompany(context.Background(), actor, created.ID, "Кристайл ООО", "kristail")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Кристайл ООО" || updated.LoginSlug != "kristail" {
		t.Fatalf("got %#v", updated)
	}
}

func TestUpdateCompanyRejectsEmptyName(t *testing.T) {
	admin, platform := newPlatformAdmin(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Кристайл", "kristail")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "admin-a", Role: identity.RoleCompanyAdmin, CompanyID: created.ID}
	if _, err := admin.UpdateCompany(context.Background(), actor, created.ID, "  ", "kristail"); !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateCompanyRejectsPurchaser(t *testing.T) {
	admin, platform := newPlatformAdmin(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Кристайл", "kristail")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "buyer", Role: identity.RolePurchaser, CompanyID: created.ID}
	if _, err := admin.UpdateCompany(context.Background(), actor, created.ID, "Other", "kristail"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestSetCompanyLogoStoresPNGForCompanyAdmin(t *testing.T) {
	admin, platform, files := newPlatformAdminWithFiles(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Кристайл", "kristail")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "admin-a", Role: identity.RoleCompanyAdmin, CompanyID: created.ID}
	updated, err := admin.SetCompanyLogo(context.Background(), actor, created.ID, png1x1)
	if err != nil {
		t.Fatal(err)
	}
	if updated.LogoContentType != "image/png" {
		t.Fatalf("content type: got %q", updated.LogoContentType)
	}
	object, ok := files.objects[identity.CompanyLogoKey(created.ID)]
	if !ok || !bytes.Equal(object.Content, png1x1) {
		t.Fatal("logo was not stored")
	}
}

func TestPublicCompanyLogoReturnsStoredBytes(t *testing.T) {
	admin, platform, _ := newPlatformAdminWithFiles(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Acme", "acme")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "admin-a", Role: identity.RoleCompanyAdmin, CompanyID: created.ID}
	if _, err := admin.SetCompanyLogo(context.Background(), actor, created.ID, png1x1); err != nil {
		t.Fatal(err)
	}
	object, err := admin.PublicCompanyLogo(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if object.ContentType != "image/png" || !bytes.Equal(object.Content, png1x1) {
		t.Fatalf("got %#v", object)
	}
}

func TestSetCompanyLogoRejectsPurchaser(t *testing.T) {
	admin, platform, _ := newPlatformAdminWithFiles(t)
	created, err := admin.CreateCompany(context.Background(), platform, "Acme", "acme")
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "buyer", Role: identity.RolePurchaser, CompanyID: created.ID}
	if _, err := admin.SetCompanyLogo(context.Background(), actor, created.ID, png1x1); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("got %v", err)
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

func newPlatformAdminWithFiles(t *testing.T) (*Admin, identity.User, *memoryFiles) {
	t.Helper()
	admin, actor := newPlatformAdmin(t)
	files := &memoryFiles{objects: map[string]port.Object{}}
	admin.files = files
	return admin, actor, files
}

type memoryFiles struct {
	objects map[string]port.Object
}

func (s *memoryFiles) Put(_ context.Context, key string, contentType string, content []byte) error {
	s.objects[key] = port.Object{Content: content, ContentType: contentType}
	return nil
}

func (s *memoryFiles) Get(_ context.Context, key string) (port.Object, error) {
	object, ok := s.objects[key]
	if !ok {
		return port.Object{}, identity.ErrNotFound
	}
	return object, nil
}

var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
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

	owner := identity.User{ID: "o1", CompanyID: "c1", Role: identity.RoleCompanyOwner}
	if _, err := lister.Execute(ctx, owner, "other"); err != nil {
		t.Fatal(err)
	}
	if repo.last.CompanyID != "c1" || repo.last.CreatedBy != "" {
		t.Fatalf("company owner filter %+v", repo.last)
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
