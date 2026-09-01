package usecase

import (
	"context"
	"testing"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func TestListJobsScopesByRole(t *testing.T) {
	repo := &listJobRepo{}
	lister := NewListJobs(repo)
	ctx := context.Background()

	purchaser := identity.User{ID: "u1", CompanyID: "c1", Role: identity.RolePurchaser}
	if _, err := lister.Execute(ctx, purchaser, "ignored"); err != nil {
		t.Fatal(err)
	}
	if repo.last.CompanyID != "c1" || repo.last.CreatedBy != "u1" {
		t.Fatalf("purchaser filter %+v", repo.last)
	}

	admin := identity.User{ID: "a1", CompanyID: "c1", Role: identity.RoleCompanyAdmin}
	if _, err := lister.Execute(ctx, admin, "other"); err != nil {
		t.Fatal(err)
	}
	if repo.last.CompanyID != "c1" || repo.last.CreatedBy != "" {
		t.Fatalf("company admin filter %+v", repo.last)
	}

	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if _, err := lister.Execute(ctx, platform, "c9"); err != nil {
		t.Fatal(err)
	}
	if repo.last.CompanyID != "c9" || repo.last.CreatedBy != "" {
		t.Fatalf("platform filter %+v", repo.last)
	}
}

type listJobRepo struct {
	last port.JobListFilter
}

func (r *listJobRepo) List(_ context.Context, filter port.JobListFilter) ([]port.JobListRow, error) {
	r.last = filter
	return []port.JobListRow{{Job: job.Job{ID: "job-1"}}}, nil
}
