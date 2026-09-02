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
		{"platform reads any", platform, otherFirm, true},
		{"legacy job only platform", purchaserA, legacy, false},
		{"legacy job platform", platform, legacy, true},
		{"company admin cannot read legacy", adminA, legacy, false},
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
	admin := identity.User{ID: "admin-a", CompanyID: "company-a", Role: identity.RoleCompanyAdmin}
	purchaser := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser}

	if !CanInviteRole(platform, identity.RoleCompanyAdmin) {
		t.Fatal("platform should invite company admin")
	}
	if CanInviteRole(platform, identity.RolePurchaser) {
		t.Fatal("platform should not invite purchaser")
	}
	if CanInviteRole(platform, identity.RolePlatformAdmin) {
		t.Fatal("nobody should invite platform admin")
	}
	if !CanInviteRole(admin, identity.RolePurchaser) || !CanInviteRole(admin, identity.RoleCompanyAdmin) {
		t.Fatal("company admin should invite staff of the firm")
	}
	if CanInviteRole(purchaser, identity.RolePurchaser) {
		t.Fatal("purchaser should not invite")
	}
}

func TestCanCreateJob(t *testing.T) {
	purchaser := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser}
	admin := identity.User{ID: "admin-a", CompanyID: "company-a", Role: identity.RoleCompanyAdmin}
	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if !CanCreateJob(purchaser) || !CanCreateJob(admin) {
		t.Fatal("company users should create jobs")
	}
	if CanCreateJob(platform) {
		t.Fatal("platform admin should not create jobs")
	}
	if CanCreateJob(identity.User{ID: "x", Role: identity.RolePurchaser}) {
		t.Fatal("purchaser without company should not create jobs")
	}
}
