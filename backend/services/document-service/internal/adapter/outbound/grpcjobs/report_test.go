package grpcjobs

import (
	"encoding/json"
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
