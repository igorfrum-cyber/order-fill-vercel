package domain

import "testing"

func TestCanInviteRole(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		actor   Role
		primary bool
		want    Role
		ok      bool
	}{
		{"primary_platform_admin_invites_platform_admin", RolePlatformAdmin, true, RolePlatformAdmin, true},
		{"secondary_platform_admin_cannot_invite_platform_admin", RolePlatformAdmin, false, RolePlatformAdmin, false},
		{"platform_admin_invites_company_owner", RolePlatformAdmin, false, RoleCompanyOwner, true},
		{"platform_admin_invites_company_admin", RolePlatformAdmin, false, RoleCompanyAdmin, true},
		{"platform_admin_invites_purchaser", RolePlatformAdmin, false, RolePurchaser, true},
		{"company_owner_invites_company_admin", RoleCompanyOwner, false, RoleCompanyAdmin, true},
		{"company_owner_invites_purchaser", RoleCompanyOwner, false, RolePurchaser, true},
		{"company_owner_cannot_invite_owner", RoleCompanyOwner, false, RoleCompanyOwner, false},
		{"company_admin_invites_purchaser", RoleCompanyAdmin, false, RolePurchaser, true},
		{"company_admin_cannot_invite_admin", RoleCompanyAdmin, false, RoleCompanyAdmin, false},
		{"company_admin_cannot_invite_owner", RoleCompanyAdmin, false, RoleCompanyOwner, false},
		{"purchaser_cannot_invite_purchaser", RolePurchaser, false, RolePurchaser, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CanInviteRole(User{Role: tc.actor, CompanyID: "co", IsPrimaryAdmin: tc.primary}, tc.want)
			if got != tc.ok {
				t.Fatalf("got %v want %v", got, tc.ok)
			}
		})
	}
}

func TestCanManageUser(t *testing.T) {
	t.Parallel()
	owner := User{ID: "o1", Role: RoleCompanyOwner, CompanyID: "co"}
	peerOwner := User{ID: "o2", Role: RoleCompanyOwner, CompanyID: "co"}
	admin := User{ID: "a1", Role: RoleCompanyAdmin, CompanyID: "co"}
	peerAdmin := User{ID: "a2", Role: RoleCompanyAdmin, CompanyID: "co"}
	buyer := User{ID: "b1", Role: RolePurchaser, CompanyID: "co"}
	other := User{ID: "b2", Role: RolePurchaser, CompanyID: "other"}
	platform := User{ID: "p1", Role: RolePlatformAdmin}
	primary := User{ID: "root", Role: RolePlatformAdmin, IsPrimaryAdmin: true}
	peerPlatform := User{ID: "p2", Role: RolePlatformAdmin}
	if !CanManageUser(platform, owner) || !CanManageUser(owner, admin) || !CanManageUser(owner, buyer) {
		t.Fatal("owner/platform should manage company staff")
	}
	if CanManageUser(admin, owner) {
		t.Fatal("company admin must not manage owner")
	}
	if !CanManageUser(admin, buyer) {
		t.Fatal("company admin should manage purchaser")
	}
	if CanManageUser(buyer, buyer) || CanManageUser(admin, other) {
		t.Fatal("purchaser and cross-company manage must fail")
	}
	if CanManageUser(owner, owner) || CanManageUser(admin, admin) || CanManageUser(platform, platform) {
		t.Fatal("actor must not manage self")
	}
	if CanManageUser(owner, peerOwner) || CanManageUser(admin, peerAdmin) {
		t.Fatal("same-role peers must not manage each other")
	}
	if !CanManageUser(primary, peerPlatform) {
		t.Fatal("primary platform admin should manage secondary platform admins")
	}
	if CanManageUser(platform, primary) || CanManageUser(platform, peerPlatform) {
		t.Fatal("secondary platform admin must not manage platform admins")
	}
}

func TestCanManageCompany(t *testing.T) {
	t.Parallel()
	if !CanManageCompany(User{Role: RolePlatformAdmin}, "co") {
		t.Fatal("platform admin")
	}
	if !CanManageCompany(User{Role: RoleCompanyOwner, CompanyID: "co"}, "co") {
		t.Fatal("owner own company")
	}
	if CanManageCompany(User{Role: RoleCompanyAdmin, CompanyID: "co"}, "other") {
		t.Fatal("admin other company")
	}
	if CanManageCompany(User{Role: RolePurchaser, CompanyID: "co"}, "co") {
		t.Fatal("purchaser must not manage company")
	}
}

func TestBoundToOwnCompany(t *testing.T) {
	t.Parallel()
	if !BoundToOwnCompany(User{Role: RoleCompanyOwner}) || !BoundToOwnCompany(User{Role: RoleCompanyAdmin}) {
		t.Fatal("keepers are bound")
	}
	if BoundToOwnCompany(User{Role: RolePurchaser}) || BoundToOwnCompany(User{Role: RolePlatformAdmin}) {
		t.Fatal("purchaser and platform admin are not bound")
	}
}
