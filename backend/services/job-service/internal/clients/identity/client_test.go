package identity

import (
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/job-service/internal/domain"
)

func TestCompanyConfigFromCompanyList(t *testing.T) {
	t.Parallel()
	companies := []*identityv1.Company{
		{Id: "other", MatchingMode: commonv1.MatchingMode_MATCHING_MODE_STANDARD},
		{Id: "co", MatchingMode: commonv1.MatchingMode_MATCHING_MODE_SMART, OrderProfile: &identityv1.CompanyOrderProfile{
			LegalName: "ООО Тест", BrandTerms: []*identityv1.CompanyBrandTerms{{Brand: "klapp", DiscountBasisPoints: 2_500, DiscountSet: true}},
		}},
	}
	got := companyConfigOf(companies, "co")
	if got.MatchingMode != domain.MatchingModeSmart || got.OrderProfile.LegalName != "ООО Тест" || len(got.OrderProfile.BrandTerms) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got := companyConfigOf(nil, "co"); got.MatchingMode != domain.MatchingModeStandard {
		t.Fatalf("missing company got %+v", got)
	}
}
