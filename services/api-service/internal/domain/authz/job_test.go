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
