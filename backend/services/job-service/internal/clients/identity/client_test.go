package identity

import (
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/job-service/internal/domain"
)

func TestMatchingModeFromCompanyList(t *testing.T) {
	t.Parallel()
	companies := []*identityv1.Company{
		{Id: "other", MatchingMode: commonv1.MatchingMode_MATCHING_MODE_STANDARD},
		{Id: "co", MatchingMode: commonv1.MatchingMode_MATCHING_MODE_SMART},
	}
	if got := matchingModeOf(companies, "co"); got != domain.MatchingModeSmart {
		t.Fatalf("got %s", got)
	}
	if got := matchingModeOf(nil, "co"); got != domain.MatchingModeStandard {
		t.Fatalf("missing company got %s", got)
	}
}
