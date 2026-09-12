package users_test

import (
	"errors"
	"testing"
	"time"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/service/users"
	"order-fill/backend/services/identity-service/internal/storage/memory"
)

func TestCreateInviteListDisableFollowRoles(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := users.New(store, func() time.Time { return now })
	if err := store.CreateCompany(t.Context(), domain.Company{ID: "co", Name: "Acme", LoginSlug: "acme", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCompany(t.Context(), domain.Company{ID: "other", Name: "Other", LoginSlug: "other", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	platform := domain.User{ID: "p1", Role: domain.RolePlatformAdmin, Login: "admin"}
	owner := domain.User{ID: "o1", Role: domain.RoleCompanyOwner, CompanyID: "co", Login: "owner"}
	admin := domain.User{ID: "a1", Role: domain.RoleCompanyAdmin, CompanyID: "co", Login: "keeper"}
	buyer := domain.User{ID: "b1", Role: domain.RolePurchaser, CompanyID: "co", Login: "buyer"}
	for _, user := range []domain.User{owner, admin, buyer} {
		if err := store.CreateUser(t.Context(), user); err != nil {
			t.Fatal(err)
		}
	}

	if _, _, err := svc.Create(t.Context(), admin, "co", "peer-admin", domain.RoleCompanyAdmin); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("admin invite admin: %v", err)
	}
	created, token, err := svc.Create(t.Context(), admin, "co", "new-buyer", domain.RolePurchaser)
	if err != nil || token == "" || created.Role != domain.RolePurchaser {
		t.Fatalf("admin invite purchaser: %+v %q %v", created, token, err)
	}
	if _, _, err := svc.Create(t.Context(), owner, "other", "x", domain.RolePurchaser); err != nil {
		t.Fatal("owner must be forced onto own company")
	}
	items, err := svc.List(t.Context(), owner, "other")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.CompanyID != "co" {
			t.Fatalf("owner listed foreign company: %+v", item)
		}
	}
	if _, err := svc.List(t.Context(), buyer, "co"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("purchaser list: %v", err)
	}
	if err := svc.Disable(t.Context(), admin, owner.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("admin disable owner: %v", err)
	}
	if err := svc.Disable(t.Context(), admin, admin.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("admin disable self: %v", err)
	}
	peerAdmin := domain.User{ID: "a2", Role: domain.RoleCompanyAdmin, CompanyID: "co", Login: "peer"}
	if err := store.CreateUser(t.Context(), peerAdmin); err != nil {
		t.Fatal(err)
	}
	if err := svc.Disable(t.Context(), admin, peerAdmin.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("admin disable peer admin: %v", err)
	}
	if err := svc.Disable(t.Context(), owner, owner.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("owner disable self: %v", err)
	}
	if err := svc.Disable(t.Context(), admin, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResetAccess(t.Context(), platform, buyer.ID); err != nil {
		t.Fatal(err)
	}
}
