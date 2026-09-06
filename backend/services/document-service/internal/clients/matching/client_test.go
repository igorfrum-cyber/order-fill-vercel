package matching

import (
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

func TestMatchingModeMapsSmart(t *testing.T) {
	t.Parallel()
	if matchingMode("smart") != commonv1.MatchingMode_MATCHING_MODE_SMART {
		t.Fatal("smart must map to MATCHING_MODE_SMART")
	}
	if matchingMode("standard") != commonv1.MatchingMode_MATCHING_MODE_STANDARD {
		t.Fatal("standard must map to MATCHING_MODE_STANDARD")
	}
}

func TestCategoryNameIncludesNotInBlank(t *testing.T) {
	t.Parallel()
	if categoryName(commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK) != orderfill.CategoryNotInBlank {
		t.Fatalf("got %q", categoryName(commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK))
	}
}
