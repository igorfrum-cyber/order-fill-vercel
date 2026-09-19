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

func TestParseChristinaProffModeDefaultsToStandard(t *testing.T) {
	t.Parallel()
	if got := ParseChristinaProffMode(""); got != ChristinaProffModeStandard {
		t.Fatalf("got %q", got)
	}
	if got := ParseChristinaProffMode("compare"); got != ChristinaProffModeCompare {
		t.Fatalf("got %q", got)
	}
	if got := ParseChristinaProffMode("fast"); got != ChristinaProffModeFast {
		t.Fatalf("got %q", got)
	}
}

func TestPlatformAdminCanSetChristinaProffMode(t *testing.T) {
	t.Parallel()
	admin := User{Role: RolePlatformAdmin}
	if !admin.CanSetChristinaProffMode() {
		t.Fatal("platform admin must set christina proff mode")
	}
	owner := User{Role: RoleCompanyOwner, CompanyID: "c1"}
	if owner.CanSetChristinaProffMode() {
		t.Fatal("owner must not set christina proff mode")
	}
}
