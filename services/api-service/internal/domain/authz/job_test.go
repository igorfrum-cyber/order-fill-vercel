package authz

import (
	"testing"

	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func TestCanAccessJob(t *testing.T) {
	companyA := "company-a"
	companyB := "company-b"
	purchaserA := identity.User{ID: "user-a", CompanyID: companyA, Role: identity.RolePurchaser}
	purchaserB := identity.User{ID: "user-b", CompanyID: companyA, Role: identity.RolePurchaser}
	adminA := identity.User{ID: "admin-a", CompanyID: companyA, Role: identity.RoleCompanyAdmin}
	adminB := identity.User{ID: "admin-b", CompanyID: companyB, Role: identity.RoleCompanyAdmin}
	ownerA := identity.User{ID: "owner-a", CompanyID: companyA, Role: identity.RoleCompanyOwner}
	ownerB := identity.User{ID: "owner-b", CompanyID: companyB, Role: identity.RoleCompanyOwner}
	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}

	own := job.Job{ID: "job-1", CompanyID: companyA, CreatedBy: purchaserA.ID}
	peer := job.Job{ID: "job-2", CompanyID: companyA, CreatedBy: purchaserB.ID}
	otherFirm := job.Job{ID: "job-3", CompanyID: companyB, CreatedBy: "user-c"}
	legacy := job.Job{ID: "job-legacy"}

	cases := []struct {
		name   string
		actor  identity.User
		entity job.Job
		want   bool
	}{
		{"purchaser owns job", purchaserA, own, true},
		{"purchaser cannot read peer", purchaserA, peer, false},
		{"company admin reads firm jobs", adminA, peer, true},
		{"company admin cannot read other firm", adminA, otherFirm, false},
		{"other firm admin cannot read", adminB, own, false},
		{"company owner reads firm jobs", ownerA, peer, true},
		{"company owner cannot read other firm", ownerA, otherFirm, false},
		{"other firm owner cannot read", ownerB, own, false},
		{"platform reads any", platform, otherFirm, true},
		{"legacy job only platform", purchaserA, legacy, false},
		{"legacy job platform", platform, legacy, true},
		{"company admin cannot read legacy", adminA, legacy, false},
		{"company owner cannot read legacy", ownerA, legacy, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := CanAccessJob(test.actor, test.entity); got != test.want {
				t.Fatalf("CanAccessJob = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCanInviteRole(t *testing.T) {
	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	owner := identity.User{ID: "owner-a", CompanyID: "company-a", Role: identity.RoleCompanyOwner}
	admin := identity.User{ID: "admin-a", CompanyID: "company-a", Role: identity.RoleCompanyAdmin}
	purchaser := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser}

	cases := []struct {
		name  string
		actor identity.User
		role  identity.Role
		want  bool
	}{
		{"platform invites owner", platform, identity.RoleCompanyOwner, true},
		{"platform invites admin", platform, identity.RoleCompanyAdmin, true},
		{"platform invites purchaser", platform, identity.RolePurchaser, true},
		{"platform cannot invite platform", platform, identity.RolePlatformAdmin, false},
		{"owner invites admin", owner, identity.RoleCompanyAdmin, true},
		{"owner invites purchaser", owner, identity.RolePurchaser, true},
		{"owner cannot invite owner", owner, identity.RoleCompanyOwner, false},
		{"owner cannot invite platform", owner, identity.RolePlatformAdmin, false},
		{"admin invites purchaser", admin, identity.RolePurchaser, true},
		{"admin cannot invite admin", admin, identity.RoleCompanyAdmin, false},
		{"admin cannot invite owner", admin, identity.RoleCompanyOwner, false},
		{"purchaser cannot invite", purchaser, identity.RolePurchaser, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := CanInviteRole(test.actor, test.role); got != test.want {
				t.Fatalf("CanInviteRole = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCanCreateJob(t *testing.T) {
	purchaser := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser}
	admin := identity.User{ID: "admin-a", CompanyID: "company-a", Role: identity.RoleCompanyAdmin}
	owner := identity.User{ID: "owner-a", CompanyID: "company-a", Role: identity.RoleCompanyOwner}
	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if !CanCreateJob(purchaser) || !CanCreateJob(admin) || !CanCreateJob(owner) {
		t.Fatal("company users should create jobs")
	}
	if CanCreateJob(platform) {
		t.Fatal("platform admin should not create jobs")
	}
	if CanCreateJob(identity.User{ID: "x", Role: identity.RolePurchaser}) {
		t.Fatal("purchaser without company should not create jobs")
	}
	if CanCreateJob(identity.User{ID: "x", Role: identity.RoleCompanyOwner}) {
		t.Fatal("owner without company should not create jobs")
	}
}

func TestCanManageCompany(t *testing.T) {
	owner := identity.User{ID: "owner-a", CompanyID: "company-a", Role: identity.RoleCompanyOwner}
	admin := identity.User{ID: "admin-a", CompanyID: "company-a", Role: identity.RoleCompanyAdmin}
	purchaser := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser}
	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}

	if !CanManageCompany(owner, "company-a") || !CanManageCompany(admin, "company-a") {
		t.Fatal("owner and admin should manage own company")
	}
	if CanManageCompany(owner, "company-b") || CanManageCompany(admin, "company-b") {
		t.Fatal("owner and admin should not manage another company")
	}
	if CanManageCompany(purchaser, "company-a") {
		t.Fatal("purchaser should not manage company")
	}
	if !CanManageCompany(platform, "company-b") {
		t.Fatal("platform should manage any company")
	}
}

func TestCanManageUser(t *testing.T) {
	owner := identity.User{ID: "owner-a", CompanyID: "company-a", Role: identity.RoleCompanyOwner}
	admin := identity.User{ID: "admin-a", CompanyID: "company-a", Role: identity.RoleCompanyAdmin}
	purchaser := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser}
	otherAdmin := identity.User{ID: "admin-b", CompanyID: "company-b", Role: identity.RoleCompanyAdmin}
	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}

	if !CanManageUser(owner, admin) || !CanManageUser(owner, purchaser) {
		t.Fatal("owner should manage company staff")
	}
	if CanManageUser(admin, owner) {
		t.Fatal("company admin should not manage owner")
	}
	if !CanManageUser(admin, purchaser) {
		t.Fatal("company admin should manage purchaser")
	}
	if !CanManageUser(platform, owner) {
		t.Fatal("platform should manage owner")
	}
	if CanManageUser(owner, otherAdmin) || CanManageUser(admin, otherAdmin) {
		t.Fatal("company staff should not manage another company")
	}
	if CanManageUser(purchaser, admin) {
		t.Fatal("purchaser should not manage users")
	}
}

func TestNeedsTwoFactorNudge(t *testing.T) {
	if !NeedsTwoFactorNudge(identity.User{Role: identity.RolePurchaser}) {
		t.Fatal("purchaser without extra login should be nudged")
	}
	if !NeedsTwoFactorNudge(identity.User{Role: identity.RoleCompanyOwner}) {
		t.Fatal("owner without extra login should be nudged")
	}
	if !NeedsTwoFactorNudge(identity.User{Role: identity.RoleCompanyAdmin}) {
		t.Fatal("admin without extra login should be nudged")
	}
	if !NeedsTwoFactorNudge(identity.User{Role: identity.RolePlatformAdmin}) {
		t.Fatal("platform admin without extra login should be nudged")
	}
	if NeedsTwoFactorNudge(identity.User{Role: identity.RolePurchaser, TwoFactorEnabled: true}) {
		t.Fatal("enabled 2FA should not be nudged")
	}
	if NeedsTwoFactorNudge(identity.User{Role: identity.RolePurchaser, HasPasskey: true}) {
		t.Fatal("a passkey should not be nudged")
	}
}
