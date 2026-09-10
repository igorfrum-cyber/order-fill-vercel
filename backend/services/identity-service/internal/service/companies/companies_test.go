package companies_test

import (
	"errors"
	"testing"
	"time"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/service/companies"
	"order-fill/backend/services/identity-service/internal/storage/memory"
)

func TestUpdateKeepsNameWhenOnlySlugSent(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := companies.New(store, func() time.Time { return now })
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	company, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeStandard)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Update(t.Context(), admin, company.ID, "", "acme-shop", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Acme" || got.LoginSlug != "acme-shop" {
		t.Fatalf("got %+v", got)
	}
}

func TestListReturnsOwnCompanyForPurchaser(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := companies.New(store, func() time.Time { return now })
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	own, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeSmart)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(t.Context(), admin, "Other", "other", domain.MatchingModeStandard); err != nil {
		t.Fatal(err)
	}
	items, err := svc.List(t.Context(), domain.User{ID: "buyer", Role: domain.RolePurchaser, CompanyID: own.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != own.ID || items[0].MatchingMode != domain.MatchingModeSmart {
		t.Fatalf("got %+v", items)
	}
}

func TestListRejectsPurchaserWithoutCompany(t *testing.T) {
	t.Parallel()
	svc := companies.New(memory.NewStore(), nil)
	_, err := svc.List(t.Context(), domain.User{ID: "buyer", Role: domain.RolePurchaser})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateSetsMatchingModeForPlatformAdmin(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := companies.New(store, func() time.Time { return now })
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	company, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeStandard)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Update(t.Context(), admin, company.ID, "", "", domain.MatchingModeSmart)
	if err != nil {
		t.Fatal(err)
	}
	if got.MatchingMode != domain.MatchingModeSmart {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdateIgnoresMatchingModeFromOwner(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := companies.New(store, func() time.Time { return now })
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	company, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeStandard)
	if err != nil {
		t.Fatal(err)
	}
	owner := domain.User{ID: "owner", Role: domain.RoleCompanyOwner, CompanyID: company.ID}
	got, err := svc.Update(t.Context(), owner, company.ID, "", "", domain.MatchingModeSmart)
	if err != nil {
		t.Fatal(err)
	}
	if got.MatchingMode != domain.MatchingModeStandard {
		t.Fatalf("owner changed matching mode: %+v", got)
	}
}
