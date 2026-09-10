package proto_test

import (
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
)

func TestReportCategoryContractNames(t *testing.T) {
	if commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION.String() != "REPORT_CATEGORY_NEEDS_DECISION" {
		t.Fatal("needs_decision category must stay stable")
	}
	if commonv1.MatchingMode_MATCHING_MODE_STANDARD.Number() != 1 {
		t.Fatal("standard mode enum number must stay stable")
	}
	if commonv1.MatchingMode_MATCHING_MODE_SMART.Number() != 2 {
		t.Fatal("smart mode enum number must stay stable")
	}
}
