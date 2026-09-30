package grpcjobs

import (
	"encoding/json"
	"math"
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

func TestCompleteReportCountsCategoriesFromRows(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal(map[string]any{
		"rows": []orderfill.ReportRow{
			{Key: "a", Status: orderfill.StatusMatched, BlankArticle: "A1"},
			{Key: "b", Status: orderfill.StatusNotInSource, BlankArticle: "B1"},
			{Key: "c", Status: orderfill.StatusLeftBlank, BlankArticle: "C1"},
			{Key: "d", Status: orderfill.StatusSourceDuplicate, BlankArticle: "D1"},
			{Key: "e", Status: orderfill.StatusNotInBlank, BlankArticle: "E1"},
			{Key: "f", Status: orderfill.StatusWarningNameDiffers, BlankArticle: "F1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	summary, rows := completeReport(raw)
	if summary.GetToOrder() != 1 || summary.GetNotInSource() != 1 || summary.GetOrderNotNeeded() != 1 {
		t.Fatalf("%+v", summary)
	}
	if summary.GetNeedsDecision() != 1 || summary.GetNotInBlank() != 1 || summary.GetCheckNameOrVolume() != 1 {
		t.Fatalf("%+v", summary)
	}
	if len(rows) != 6 || rows[3].GetCategory() != commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION {
		t.Fatalf("%+v", rows)
	}
}

func TestCompleteReportPrefersCanonicalCategory(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal(map[string]any{
		"rows": []orderfill.ReportRow{
			{Key: "a", Status: orderfill.StatusLeftBlank, Category: orderfill.CategoryCheckNameOrVolume},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	summary, rows := completeReport(raw)
	if summary.GetCheckNameOrVolume() != 1 || summary.GetOrderNotNeeded() != 0 {
		t.Fatalf("%+v", summary)
	}
	if rows[0].GetCategory() != commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME {
		t.Fatalf("%+v", rows)
	}
}

func TestCompleteReportUsesSummaryWhenThereAreNoRows(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(map[string]any{"summary": orderfill.Summary{
		Unmatched: 2, Suspicious: 3, NotInBlank: 4, Filled: 5, LeftBlank: 6, Duplicates: 7,
	}})
	if err != nil {
		t.Fatal(err)
	}
	summary, rows := completeReport(raw)
	if len(rows) != 0 || summary.GetNotInSource() != 2 || summary.GetCheckNameOrVolume() != 3 ||
		summary.GetNotInBlank() != 4 || summary.GetToOrder() != 5 || summary.GetOrderNotNeeded() != 6 || summary.GetNeedsDecision() != 7 {
		t.Fatalf("summary=%+v rows=%+v", summary, rows)
	}
}

func TestCompleteReportRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	if summary, rows := completeReport([]byte("{")); summary != nil || rows != nil {
		t.Fatalf("malformed report = %+v, %+v; want nil values", summary, rows)
	}
}

func TestInt32ClampBoundaries(t *testing.T) {
	t.Parallel()

	if got := int32Clamp(math.MaxInt32 + 1); got != math.MaxInt32 {
		t.Fatalf("upper clamp = %d", got)
	}
	if got := int32Clamp(math.MinInt32 - 1); got != math.MinInt32 {
		t.Fatalf("lower clamp = %d", got)
	}
	if got := int32Clamp(12); got != 12 {
		t.Fatalf("in-range value = %d", got)
	}
}

func TestCategoryOfSupportsCanonicalAndLegacyStatuses(t *testing.T) {
	t.Parallel()

	canonical := []struct {
		name     string
		category string
		want     commonv1.ReportCategory
	}{
		{"needs decision", orderfill.CategoryNeedsDecision, commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION},
		{"not in source", orderfill.CategoryNotInSource, commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE},
		{"check name or volume", orderfill.CategoryCheckNameOrVolume, commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME},
		{"not in blank", orderfill.CategoryNotInBlank, commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK},
		{"order not needed", orderfill.CategoryOrderNotNeeded, commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED},
		{"to order", orderfill.CategoryToOrder, commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER},
	}
	for _, tc := range canonical {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := categoryOf(orderfill.ReportRow{Category: tc.category}); got != tc.want {
				t.Fatalf("categoryOf() = %v, want %v", got, tc.want)
			}
		})
	}

	legacy := []struct {
		name      string
		status    string
		duplicate bool
		want      commonv1.ReportCategory
	}{
		{"not in source", orderfill.StatusNotInSource, false, commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE},
		{"not in blank", orderfill.StatusNotInBlank, false, commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK},
		{"name differs", orderfill.StatusWarningNameDiffers, false, commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME},
		{"name only", orderfill.StatusWarningNameOnly, false, commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION},
		{"source duplicate", orderfill.StatusSourceDuplicate, false, commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION},
		{"left blank", orderfill.StatusLeftBlank, false, commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED},
		{"duplicate fallback", "unknown", true, commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION},
		{"to order fallback", "unknown", false, commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER},
	}
	for _, tc := range legacy {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := orderfill.ReportRow{Status: tc.status, Duplicate: tc.duplicate}
			if got := categoryOf(row); got != tc.want {
				t.Fatalf("categoryOf(%+v) = %v, want %v", row, got, tc.want)
			}
		})
	}
}
