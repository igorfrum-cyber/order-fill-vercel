package grpcjobs

import (
	"encoding/json"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

func completeReport(raw []byte) (*jobsv1.ReportSummary, []*jobsv1.ReportRow) {
	var payload struct {
		Summary orderfill.Summary     `json:"summary"`
		Rows    []orderfill.ReportRow `json:"rows"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return nil, nil
	}
	rows := make([]*jobsv1.ReportRow, 0, len(payload.Rows))
	summary := &jobsv1.ReportSummary{
		NotInSource:       int32(payload.Summary.Unmatched),
		CheckNameOrVolume: int32(payload.Summary.Suspicious),
		NotInBlank:        int32(payload.Summary.NotInBlank),
		ToOrder:           int32(payload.Summary.Filled),
		OrderNotNeeded:    int32(payload.Summary.LeftBlank),
		NeedsDecision:     int32(payload.Summary.Duplicates),
	}
	if len(payload.Rows) == 0 {
		return summary, rows
	}
	summary = &jobsv1.ReportSummary{}
	for _, row := range payload.Rows {
		cat := categoryOf(row)
		rows = append(rows, &jobsv1.ReportRow{
			Id: row.Key, Category: cat, Article: row.BlankArticle, Name: row.BlankName,
		})
		switch cat {
		case commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION:
			summary.NeedsDecision++
		case commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE:
			summary.NotInSource++
		case commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME:
			summary.CheckNameOrVolume++
		case commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK:
			summary.NotInBlank++
		case commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED:
			summary.OrderNotNeeded++
		default:
			summary.ToOrder++
		}
	}
	return summary, rows
}

func categoryOf(row orderfill.ReportRow) commonv1.ReportCategory {
	switch row.Status {
	case orderfill.StatusNotInSource:
		return commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE
	case orderfill.StatusNotInBlank:
		return commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK
	case orderfill.StatusWarningNameDiffers:
		return commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME
	case orderfill.StatusWarningNameOnly, orderfill.StatusSourceDuplicate:
		return commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION
	case orderfill.StatusLeftBlank:
		return commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED
	default:
		if row.Duplicate {
			return commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION
		}
		return commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER
	}
}
