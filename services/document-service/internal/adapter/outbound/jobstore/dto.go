package jobstore

import (
	"encoding/json"
	"fmt"

	"order-fill/services/document-service/internal/app/port"
	"order-fill/services/document-service/internal/domain/orderfill"
)

// The DTOs below own the JSONB wire format of the jobs.output_files and
// job_reports columns. api-service reads these columns through its own
// independent DTOs, so the snake_case names are a cross-service contract: they
// must not be renamed together with the domain fields.

type outputFileDTO struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	StorageKey  string `json:"storage_key"`
}

type summaryDTO struct {
	Brand                  string  `json:"brand"`
	OrderMonthLabel        string  `json:"order_month_label"`
	AdjustmentLabel        string  `json:"adjustment_label"`
	ActualMainPeriod       string  `json:"actual_main_period"`
	ActualPreviousPeriod   string  `json:"actual_previous_period"`
	SourceCity             string  `json:"source_city"`
	CityRule               string  `json:"city_rule"`
	DeliveryWeeks          float64 `json:"delivery_weeks"`
	Filled                 int     `json:"filled"`
	LeftBlank              int     `json:"left_blank"`
	Suspicious             int     `json:"suspicious"`
	Unmatched              int     `json:"unmatched"`
	Duplicates             int     `json:"duplicates"`
	BlankDuplicateArticles int     `json:"blank_duplicate_articles"`
	SourceItems            int     `json:"source_items"`
	SourceArticles         int     `json:"source_articles"`
	SourceSheet            string  `json:"source_sheet"`
	SourceHeaderRow        int     `json:"source_header_row"`
	BlankSheet             string  `json:"blank_sheet"`
	BlankHeaderRow         int     `json:"blank_header_row"`
}

type reportRowDTO struct {
	Key                 string                  `json:"key"`
	Status              string                  `json:"status"`
	BlankID             string                  `json:"blank_id"`
	BlankLabel          string                  `json:"blank_label"`
	BlankRow            int                     `json:"blank_row"`
	BlankQuantityColumn int                     `json:"blank_quantity_col"`
	BlankArticle        string                  `json:"blank_article"`
	BlankName           string                  `json:"blank_name"`
	BlankUnit           string                  `json:"blank_unit"`
	BlankBoxSize        string                  `json:"blank_box_size"`
	SourceRow           *int                    `json:"source_row"`
	SourceArticle       string                  `json:"source_article"`
	SourceName          string                  `json:"source_name"`
	HasOrderedFact      bool                    `json:"has_ordered_fact"`
	OrderedFact         *float64                `json:"ordered_fact"`
	SourceComment       string                  `json:"source_comment"`
	Stock               string                  `json:"stock"`
	InTransit           string                  `json:"in_transit"`
	Recommended         *float64                `json:"recommended"`
	Rounded             *int                    `json:"rounded"`
	BaseRounded         *int                    `json:"base_rounded"`
	Inserted            *float64                `json:"inserted"`
	AutoComment         string                  `json:"auto_comment"`
	AdjustmentLabel     string                  `json:"adjustment_label"`
	BoxAdjusted         bool                    `json:"box_adjusted"`
	Duplicate           bool                    `json:"duplicate"`
	DuplicateCandidates []duplicateCandidateDTO `json:"duplicate_candidates"`
	Editable            bool                    `json:"editable"`
	Similarity          float64                 `json:"similarity"`
}

type duplicateCandidateDTO struct {
	SourceRow     int     `json:"source_row"`
	SourceArticle string  `json:"source_article"`
	SourceName    string  `json:"source_name"`
	Recommended   float64 `json:"recommended"`
	Rounded       int     `json:"rounded"`
	Stock         string  `json:"stock"`
	InTransit     string  `json:"in_transit"`
}

func outputFilesToDTO(files []port.OutputFile) []outputFileDTO {
	result := make([]outputFileDTO, 0, len(files))
	for _, file := range files {
		result = append(result, outputFileDTO{
			ID:          file.ID,
			Label:       file.Label,
			Name:        file.Name,
			ContentType: file.ContentType,
			SizeBytes:   file.SizeBytes,
			StorageKey:  file.StorageKey,
		})
	}
	return result
}

func outputFilesToDomain(files []outputFileDTO) []port.OutputFile {
	result := make([]port.OutputFile, 0, len(files))
	for _, file := range files {
		result = append(result, port.OutputFile{
			ID:          file.ID,
			Label:       file.Label,
			Name:        file.Name,
			ContentType: file.ContentType,
			SizeBytes:   file.SizeBytes,
			StorageKey:  file.StorageKey,
		})
	}
	return result
}

func summaryToDTO(summary orderfill.Summary) summaryDTO {
	return summaryDTO{
		Brand:                  summary.Brand,
		OrderMonthLabel:        summary.OrderMonthLabel,
		AdjustmentLabel:        summary.AdjustmentLabel,
		ActualMainPeriod:       summary.ActualMainPeriod,
		ActualPreviousPeriod:   summary.ActualPreviousPeriod,
		SourceCity:             summary.SourceCity,
		CityRule:               summary.CityRule,
		DeliveryWeeks:          summary.DeliveryWeeks,
		Filled:                 summary.Filled,
		LeftBlank:              summary.LeftBlank,
		Suspicious:             summary.Suspicious,
		Unmatched:              summary.Unmatched,
		Duplicates:             summary.Duplicates,
		BlankDuplicateArticles: summary.BlankDuplicateArticles,
		SourceItems:            summary.SourceItems,
		SourceArticles:         summary.SourceArticles,
		SourceSheet:            summary.SourceSheet,
		SourceHeaderRow:        summary.SourceHeaderRow,
		BlankSheet:             summary.BlankSheet,
		BlankHeaderRow:         summary.BlankHeaderRow,
	}
}

func summaryToDomain(summary summaryDTO) orderfill.Summary {
	return orderfill.Summary{
		Brand:                  summary.Brand,
		OrderMonthLabel:        summary.OrderMonthLabel,
		AdjustmentLabel:        summary.AdjustmentLabel,
		ActualMainPeriod:       summary.ActualMainPeriod,
		ActualPreviousPeriod:   summary.ActualPreviousPeriod,
		SourceCity:             summary.SourceCity,
		CityRule:               summary.CityRule,
		DeliveryWeeks:          summary.DeliveryWeeks,
		Filled:                 summary.Filled,
		LeftBlank:              summary.LeftBlank,
		Suspicious:             summary.Suspicious,
		Unmatched:              summary.Unmatched,
		Duplicates:             summary.Duplicates,
		BlankDuplicateArticles: summary.BlankDuplicateArticles,
		SourceItems:            summary.SourceItems,
		SourceArticles:         summary.SourceArticles,
		SourceSheet:            summary.SourceSheet,
		SourceHeaderRow:        summary.SourceHeaderRow,
		BlankSheet:             summary.BlankSheet,
		BlankHeaderRow:         summary.BlankHeaderRow,
	}
}

func reportRowsToDTO(rows []orderfill.ReportRow) []reportRowDTO {
	result := make([]reportRowDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, reportRowDTO{
			Key:                 row.Key,
			Status:              row.Status,
			BlankID:             row.BlankID,
			BlankLabel:          row.BlankLabel,
			BlankRow:            row.BlankRow,
			BlankQuantityColumn: row.BlankQuantityColumn,
			BlankArticle:        row.BlankArticle,
			BlankName:           row.BlankName,
			BlankUnit:           row.BlankUnit,
			BlankBoxSize:        row.BlankBoxSize,
			SourceRow:           row.SourceRow,
			SourceArticle:       row.SourceArticle,
			SourceName:          row.SourceName,
			HasOrderedFact:      row.HasOrderedFact,
			OrderedFact:         row.OrderedFact,
			SourceComment:       row.SourceComment,
			Stock:               row.Stock,
			InTransit:           row.InTransit,
			Recommended:         row.Recommended,
			Rounded:             row.Rounded,
			BaseRounded:         row.BaseRounded,
			Inserted:            row.Inserted,
			AutoComment:         row.AutoComment,
			AdjustmentLabel:     row.AdjustmentLabel,
			BoxAdjusted:         row.BoxAdjusted,
			Duplicate:           row.Duplicate,
			DuplicateCandidates: duplicateCandidatesToDTO(row.DuplicateCandidates),
			Editable:            row.Editable,
			Similarity:          row.Similarity,
		})
	}
	return result
}

func reportRowsToDomain(rows []reportRowDTO) []orderfill.ReportRow {
	result := make([]orderfill.ReportRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, orderfill.ReportRow{
			Key:                 row.Key,
			Status:              row.Status,
			BlankID:             row.BlankID,
			BlankLabel:          row.BlankLabel,
			BlankRow:            row.BlankRow,
			BlankQuantityColumn: row.BlankQuantityColumn,
			BlankArticle:        row.BlankArticle,
			BlankName:           row.BlankName,
			BlankUnit:           row.BlankUnit,
			BlankBoxSize:        row.BlankBoxSize,
			SourceRow:           row.SourceRow,
			SourceArticle:       row.SourceArticle,
			SourceName:          row.SourceName,
			HasOrderedFact:      row.HasOrderedFact,
			OrderedFact:         row.OrderedFact,
			SourceComment:       row.SourceComment,
			Stock:               row.Stock,
			InTransit:           row.InTransit,
			Recommended:         row.Recommended,
			Rounded:             row.Rounded,
			BaseRounded:         row.BaseRounded,
			Inserted:            row.Inserted,
			AutoComment:         row.AutoComment,
			AdjustmentLabel:     row.AdjustmentLabel,
			BoxAdjusted:         row.BoxAdjusted,
			Duplicate:           row.Duplicate,
			DuplicateCandidates: duplicateCandidatesToDomain(row.DuplicateCandidates),
			Editable:            row.Editable,
			Similarity:          row.Similarity,
		})
	}
	return result
}

func duplicateCandidatesToDTO(candidates []orderfill.DuplicateCandidate) []duplicateCandidateDTO {
	result := make([]duplicateCandidateDTO, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, duplicateCandidateDTO{
			SourceRow:     candidate.SourceRow,
			SourceArticle: candidate.SourceArticle,
			SourceName:    candidate.SourceName,
			Recommended:   candidate.Recommended,
			Rounded:       candidate.Rounded,
			Stock:         candidate.Stock,
			InTransit:     candidate.InTransit,
		})
	}
	return result
}

func duplicateCandidatesToDomain(candidates []duplicateCandidateDTO) []orderfill.DuplicateCandidate {
	result := make([]orderfill.DuplicateCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, orderfill.DuplicateCandidate{
			SourceRow:     candidate.SourceRow,
			SourceArticle: candidate.SourceArticle,
			SourceName:    candidate.SourceName,
			Recommended:   candidate.Recommended,
			Rounded:       candidate.Rounded,
			Stock:         candidate.Stock,
			InTransit:     candidate.InTransit,
		})
	}
	return result
}

func marshalJSON(value any, column string) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", column, err)
	}
	return encoded, nil
}

func unmarshalJSON(raw []byte, target any, column string) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal %s: %w", column, err)
	}
	return nil
}
