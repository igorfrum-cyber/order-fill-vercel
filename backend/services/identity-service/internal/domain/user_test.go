package domain

import (
	"testing"
	"time"
)

func TestUserDisabledAndRoleCapabilities(t *testing.T) {
	t.Parallel()

	if (User{}).Disabled() {
		t.Fatal("zero user should be enabled")
	}
	if (User{CompanyDisabled: true}).Disabled() != true {
		t.Fatal("disabled company should disable its user")
	}
	disabledAt := time.Now()
	admin := User{Role: RolePlatformAdmin}
	if !admin.CanSetMatchingMode() || !admin.CanSetChristinaProffMode() {
		t.Fatal("active platform admin should be allowed to configure matching")
	}
	admin.DisabledAt = &disabledAt
	if admin.CanSetMatchingMode() || admin.CanSetChristinaProffMode() {
		t.Fatal("disabled platform admin should not configure matching")
	}
	if (User{Role: RoleCompanyOwner}).CanSetMatchingMode() {
		t.Fatal("company owner should not configure matching")
	}
}

func TestCompanyDisabledAndHasLogo(t *testing.T) {
	t.Parallel()

	company := Company{}
	if company.Disabled() || company.HasLogo() {
		t.Fatal("empty company should be enabled and have no logo")
	}
	company.LogoContentType = "image/png"
	disabledAt := time.Now()
	company.DisabledAt = &disabledAt
	if !company.Disabled() || !company.HasLogo() {
		t.Fatalf("company flags = disabled:%t logo:%t", company.Disabled(), company.HasLogo())
	}
}
