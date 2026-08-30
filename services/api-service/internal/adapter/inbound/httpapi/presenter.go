package httpapi

import (
	"fmt"
	"time"

	"order-fill/services/api-service/internal/domain/job"
	"order-fill/services/api-service/internal/domain/preview"
)

// The DTOs below are the wire contract described in packages/contracts/openapi.yaml.
// They are deliberately separate from the domain types so the transport can
// evolve without touching business rules.

type jobResponse struct {
	ID              string               `json:"id"`
	Type            string               `json:"type"`
	Status          string               `json:"status"`
	Brand           string               `json:"brand,omitempty"`
	OrderMonth      string               `json:"order_month,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Error           *errorResponse       `json:"error,omitempty"`
	InputFiles      []inputFileResponse  `json:"input_files"`
	OutputFiles     []outputFileResponse `json:"output_files"`
	Progress        float64              `json:"progress"`
	ProgressMessage string               `json:"progress_message,omitempty"`
}

type inputFileResponse struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type outputFileResponse struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Name         string `json:"name"`
	ContentType  string `json:"content_type"`
	SizeBytes    int64  `json:"size_bytes"`
	DownloadPath string `json:"download_path"`
}

type reportResponse struct {
	JobID   string              `json:"job_id"`
	Summary summaryResponse     `json:"summary"`
	Rows    []reportRowResponse `json:"rows"`
}

type summaryResponse struct {
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
	NotInBlank             int     `json:"not_in_blank"`
	BlankDuplicateArticles int     `json:"blank_duplicate_articles"`
	SourceItems            int     `json:"source_items"`
	SourceArticles         int     `json:"source_articles"`
	SourceSheet            string  `json:"source_sheet"`
	SourceHeaderRow        int     `json:"source_header_row"`
	BlankSheet             string  `json:"blank_sheet"`
	BlankHeaderRow         int     `json:"blank_header_row"`
}

type reportRowResponse struct {
	Key                 string                       `json:"key"`
	Status              string                       `json:"status"`
	BlankID             string                       `json:"blank_id"`
	BlankLabel          string                       `json:"blank_label"`
	BlankRow            int                          `json:"blank_row"`
	BlankQuantityColumn int                          `json:"blank_quantity_col"`
	BlankArticle        string                       `json:"blank_article"`
	BlankName           string                       `json:"blank_name"`
	BlankUnit           string                       `json:"blank_unit"`
	BlankBoxSize        string                       `json:"blank_box_size"`
	SourceRow           *int                         `json:"source_row"`
	SourceArticle       string                       `json:"source_article"`
	SourceName          string                       `json:"source_name"`
	HasOrderedFact      bool                         `json:"has_ordered_fact"`
	OrderedFact         *float64                     `json:"ordered_fact"`
	SourceComment       string                       `json:"source_comment"`
	Stock               string                       `json:"stock"`
	InTransit           string                       `json:"in_transit"`
	Recommended         *float64                     `json:"recommended"`
	Rounded             *int                         `json:"rounded"`
	BaseRounded         *int                         `json:"base_rounded"`
	Inserted            *float64                     `json:"inserted"`
	AutoComment         string                       `json:"auto_comment"`
	AdjustmentLabel     string                       `json:"adjustment_label"`
	BoxAdjusted         bool                         `json:"box_adjusted"`
	Duplicate           bool                         `json:"duplicate"`
	DuplicateCandidates []duplicateCandidateResponse `json:"duplicate_candidates"`
	Editable            bool                         `json:"editable"`
	Similarity          float64                      `json:"similarity"`
}

type duplicateCandidateResponse struct {
	SourceRow     int     `json:"source_row"`
	SourceArticle string  `json:"source_article"`
	SourceName    string  `json:"source_name"`
	Recommended   float64 `json:"recommended"`
	Rounded       int     `json:"rounded"`
	Stock         string  `json:"stock"`
	InTransit     string  `json:"in_transit"`
}

type errorResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func presentJob(entity job.Job) jobResponse {
	response := jobResponse{
		ID:              entity.ID,
		Type:            string(entity.Type),
		Status:          string(entity.Status),
		Brand:           entity.Brand,
		OrderMonth:      entity.OrderMonth,
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
		InputFiles:      make([]inputFileResponse, 0, len(entity.InputFiles)),
		OutputFiles:     presentOutputFiles(entity.ID, entity.OutputFiles),
		Progress:        entity.Progress,
		ProgressMessage: entity.ProgressMessage,
	}
	if entity.Failure != nil {
		response.Error = &errorResponse{Code: entity.Failure.Code, Message: entity.Failure.Message}
	}
	for _, file := range entity.InputFiles {
		response.InputFiles = append(response.InputFiles, inputFileResponse{
			ID:          file.ID,
			Role:        string(file.Role),
			Name:        file.Name,
			ContentType: file.ContentType,
			SizeBytes:   file.SizeBytes,
		})
	}
	return response
}

func presentOutputFiles(jobID string, files []job.OutputFile) []outputFileResponse {
	response := make([]outputFileResponse, 0, len(files))
	for _, file := range files {
		response = append(response, outputFileResponse{
			ID:           file.ID,
			Label:        file.Label,
			Name:         file.Name,
			ContentType:  file.ContentType,
			SizeBytes:    file.SizeBytes,
			DownloadPath: fmt.Sprintf("/api/v1/jobs/%s/files/%s", jobID, file.ID),
		})
	}
	return response
}

func presentReport(report job.Report) reportResponse {
	rows := make([]reportRowResponse, 0, len(report.Rows))
	for _, row := range report.Rows {
		candidates := make([]duplicateCandidateResponse, 0, len(row.DuplicateCandidates))
		for _, candidate := range row.DuplicateCandidates {
			candidates = append(candidates, duplicateCandidateResponse(candidate))
		}
		rows = append(rows, reportRowResponse{
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
			DuplicateCandidates: candidates,
			Editable:            row.Editable,
			Similarity:          row.Similarity,
		})
	}
	return reportResponse{
		JobID:   report.JobID,
		Summary: summaryResponse(report.Summary),
		Rows:    rows,
	}
}

func presentPreviewMeta(meta preview.Meta) previewMetaResponse {
	sheets := make([]previewSheetResponse, 0, len(meta.Sheets))
	for _, sheet := range meta.Sheets {
		sheets = append(sheets, previewSheetResponse{
			Name:          sheet.Name,
			Index:         sheet.Index,
			MaxRow:        sheet.MaxRow,
			MaxColumn:     sheet.MaxColumn,
			HeaderRow:     sheet.HeaderRow,
			ArticleColumn: sheet.ArticleColumn,
		})
	}
	return previewMetaResponse{ChunkRows: meta.ChunkRows, Sheets: sheets}
}

func presentPreviewWindow(sheetIndex int, window preview.Window) previewWindowResponse {
	return previewWindowResponse{
		SheetIndex: sheetIndex,
		FromRow:    window.FromRow,
		ToRow:      window.ToRow,
		Rows:       window.Rows,
	}
}

func presentPreviewHit(sheetIndex int, hit preview.Hit) previewHitResponse {
	return previewHitResponse{
		Found:      hit.Found,
		Row:        hit.Row,
		Column:     hit.Column,
		SheetIndex: sheetIndex,
	}
}

type previewMetaResponse struct {
	ChunkRows int                    `json:"chunk_rows"`
	Sheets    []previewSheetResponse `json:"sheets"`
}

type previewSheetResponse struct {
	Name          string `json:"name"`
	Index         int    `json:"index"`
	MaxRow        int    `json:"max_row"`
	MaxColumn     int    `json:"max_column"`
	HeaderRow     int    `json:"header_row,omitempty"`
	ArticleColumn int    `json:"article_column,omitempty"`
}

type previewWindowResponse struct {
	SheetIndex int        `json:"sheet_index"`
	FromRow    int        `json:"from_row"`
	ToRow      int        `json:"to_row"`
	Rows       [][]string `json:"rows"`
}

type previewHitResponse struct {
	Found      bool `json:"found"`
	Row        int  `json:"row,omitempty"`
	Column     int  `json:"column,omitempty"`
	SheetIndex int  `json:"sheet_index"`
}
