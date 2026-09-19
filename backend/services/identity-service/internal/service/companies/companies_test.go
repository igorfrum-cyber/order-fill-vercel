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
	company, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeStandard, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Update(t.Context(), admin, company.ID, "", "acme-shop", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Acme" || got.LoginSlug != "acme-shop" {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdateOrderProfileNormalizesAndScopesCompany(t *testing.T) {
	t.Parallel()
	store := memory.NewStore()
	svc := companies.New(store, nil)
	platform := domain.User{ID: "platform", Role: domain.RolePlatformAdmin}
	company, err := svc.Create(t.Context(), platform, "Acme", "acme-profile", domain.MatchingModeStandard, "")
	if err != nil {
		t.Fatal(err)
	}
	owner := domain.User{ID: "owner", Role: domain.RoleCompanyOwner, CompanyID: company.ID}
	got, err := svc.UpdateOrderProfile(t.Context(), owner, "ignored", domain.OrderProfile{
		LegalName:  "  ООО Тест  ",
		BrandTerms: []domain.BrandTerms{{Brand: "ANGIOPHARM", DiscountBasisPoints: 3_525, DiscountSet: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != company.ID || got.OrderProfile.LegalName != "ООО Тест" || got.OrderProfile.BrandTerms[0].Brand != "angiopharm" {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdateOrderProfileRejectsPurchaserAndInvalidDiscount(t *testing.T) {
	t.Parallel()
	store := memory.NewStore()
	svc := companies.New(store, nil)
	platform := domain.User{ID: "platform", Role: domain.RolePlatformAdmin}
	company, err := svc.Create(t.Context(), platform, "Acme", "acme-invalid-profile", domain.MatchingModeStandard, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateOrderProfile(t.Context(), domain.User{ID: "buyer", Role: domain.RolePurchaser, CompanyID: company.ID}, company.ID, domain.OrderProfile{})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("purchaser got %v", err)
	}
	_, err = svc.UpdateOrderProfile(t.Context(), platform, company.ID, domain.OrderProfile{
		BrandTerms: []domain.BrandTerms{{Brand: "klapp", DiscountBasisPoints: 10_001, DiscountSet: true}},
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("discount got %v", err)
	}
}

func TestListReturnsOwnCompanyForPurchaser(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := companies.New(store, func() time.Time { return now })
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	own, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeSmart, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(t.Context(), admin, "Other", "other", domain.MatchingModeStandard, ""); err != nil {
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
	company, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeStandard, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Update(t.Context(), admin, company.ID, "", "", domain.MatchingModeSmart, "")
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
	company, err := svc.Create(t.Context(), admin, "Acme", "acme", domain.MatchingModeStandard, "")
	if err != nil {
		t.Fatal(err)
	}
	owner := domain.User{ID: "owner", Role: domain.RoleCompanyOwner, CompanyID: company.ID}
	got, err := svc.Update(t.Context(), owner, company.ID, "", "", domain.MatchingModeSmart, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.MatchingMode != domain.MatchingModeStandard {
		t.Fatalf("owner changed matching mode: %+v", got)
	}
}

func TestCreateDefaultsChristinaProffModeToStandard(t *testing.T) {
	t.Parallel()
	svc := companies.New(memory.NewStore(), nil)
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	got, err := svc.Create(t.Context(), admin, "Acme", "acme-default-proff", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ChristinaProffMode != domain.ChristinaProffModeStandard {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdateSetsChristinaProffModeForPlatformAdmin(t *testing.T) {
	t.Parallel()
	svc := companies.New(memory.NewStore(), nil)
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	company, err := svc.Create(t.Context(), admin, "Acme", "acme-proff", domain.MatchingModeStandard, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Update(t.Context(), admin, company.ID, "", "", "", domain.ChristinaProffModeCompare)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChristinaProffMode != domain.ChristinaProffModeCompare {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdateIgnoresChristinaProffModeFromOwner(t *testing.T) {
	t.Parallel()
	svc := companies.New(memory.NewStore(), nil)
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	company, err := svc.Create(t.Context(), admin, "Acme", "acme-proff-owner", domain.MatchingModeStandard, "")
	if err != nil {
		t.Fatal(err)
	}
	owner := domain.User{ID: "owner", Role: domain.RoleCompanyOwner, CompanyID: company.ID}
	got, err := svc.Update(t.Context(), owner, company.ID, "", "", "", domain.ChristinaProffModeCompare)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChristinaProffMode != domain.ChristinaProffModeStandard {
		t.Fatalf("owner changed christina proff mode: %+v", got)
	}
}
