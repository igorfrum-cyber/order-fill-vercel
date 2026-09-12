package domain

import "testing"

func TestCanInviteRole(t *testing.T) {
	t.Parallel()
	cases := []struct {
		actor Role
		want  Role
		ok    bool
	}{
		{RolePlatformAdmin, RoleCompanyOwner, true},
		{RolePlatformAdmin, RoleCompanyAdmin, true},
		{RolePlatformAdmin, RolePurchaser, true},
		{RolePlatformAdmin, RolePlatformAdmin, false},
		{RoleCompanyOwner, RoleCompanyAdmin, true},
		{RoleCompanyOwner, RolePurchaser, true},
		{RoleCompanyOwner, RoleCompanyOwner, false},
		{RoleCompanyAdmin, RolePurchaser, true},
		{RoleCompanyAdmin, RoleCompanyAdmin, false},
		{RoleCompanyAdmin, RoleCompanyOwner, false},
		{RolePurchaser, RolePurchaser, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.actor)+"/"+string(tc.want), func(t *testing.T) {
			t.Parallel()
			got := CanInviteRole(User{Role: tc.actor, CompanyID: "co"}, tc.want)
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
