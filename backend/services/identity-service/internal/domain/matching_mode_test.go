package domain

import "testing"

func TestParseMatchingModeDefaultsToStandard(t *testing.T) {
	t.Parallel()
	if got := ParseMatchingMode(""); got != MatchingModeStandard {
		t.Fatalf("got %q", got)
	}
	if got := ParseMatchingMode("smart"); got != MatchingModeSmart {
		t.Fatalf("got %q", got)
	}
}

func TestPlatformAdminCanSetMatchingMode(t *testing.T) {
	t.Parallel()
	admin := User{Role: RolePlatformAdmin}
	if !admin.CanSetMatchingMode() {
		t.Fatal("platform admin must set matching mode")
	}
	owner := User{Role: RoleCompanyOwner, CompanyID: "c1"}
	if owner.CanSetMatchingMode() {
		t.Fatal("owner must not set matching mode yet")
	}
}
