package postgres

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"order-fill/services/api-service/internal/domain/job"
)

func TestInputFilesRoundTrip(t *testing.T) {
	files := []job.InputFile{{
		ID:          "file-1",
		Role:        job.RoleSource,
		Name:        "source.xlsx",
		ContentType: "application/vnd.ms-excel",
		SizeBytes:   1024,
		StorageKey:  "jobs/job-1/inputs/0-source.xlsx",
	}}

	encoded, err := json.Marshal(inputFilesToDTO(files))
	if err != nil {
		t.Fatalf("marshal input files: %v", err)
	}
	assertKeys(t, firstObject(t, encoded), []string{"id", "role", "name", "content_type", "size_bytes", "storage_key"})

	var decoded []inputFileDTO
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal input files: %v", err)
	}
	if got := inputFilesToDomain(decoded); !reflect.DeepEqual(got, files) {
		t.Fatalf("input files round trip mismatch: got %+v want %+v", got, files)
	}
}

func TestOutputFilesRoundTrip(t *testing.T) {
	files := []job.OutputFile{{
		ID:          "out-1",
		Label:       "Заполненный бланк",
		Name:        "filled.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		SizeBytes:   2048,
		StorageKey:  "jobs/job-1/outputs/filled.xlsx",
	}}

	encoded, err := json.Marshal(outputFilesToDTO(files))
	if err != nil {
		t.Fatalf("marshal output files: %v", err)
	}
	assertKeys(t, firstObject(t, encoded), []string{"id", "label", "name", "content_type", "size_bytes", "storage_key"})

	var decoded []outputFileDTO
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal output files: %v", err)
	}
	if got := outputFilesToDomain(decoded); !reflect.DeepEqual(got, files) {
		t.Fatalf("output files round trip mismatch: got %+v want %+v", got, files)
	}
}

func TestEmptyFileListsMarshalAsEmptyArrays(t *testing.T) {
	inputs, err := json.Marshal(inputFilesToDTO(nil))
	if err != nil {
		t.Fatalf("marshal nil input files: %v", err)
	}
	if string(inputs) != "[]" {
		t.Fatalf("nil input files: got %s want []", inputs)
	}
	outputs, err := json.Marshal(outputFilesToDTO([]job.OutputFile{}))
	if err != nil {
		t.Fatalf("marshal empty output files: %v", err)
	}
	if string(outputs) != "[]" {
		t.Fatalf("empty output files: got %s want []", outputs)
	}

	var decodedInputs []inputFileDTO
	if err := json.Unmarshal(inputs, &decodedInputs); err != nil {
		t.Fatalf("unmarshal empty input files: %v", err)
	}
	if got := inputFilesToDomain(decodedInputs); len(got) != 0 || got == nil {
		t.Fatalf("empty input files should decode to an empty non-nil slice, got %#v", got)
	}
}

func TestSummaryRoundTrip(t *testing.T) {
	summary := job.Summary{
		Brand:                  "north",
		OrderMonthLabel:        "Сентябрь 2026",
		AdjustmentLabel:        "+10%",
		ActualMainPeriod:       "2026-09",
		ActualPreviousPeriod:   "2026-08",
		SourceCity:             "Москва",
		CityRule:               "moscow",
		DeliveryWeeks:          2.5,
		Filled:                 12,
		LeftBlank:              3,
		Suspicious:             1,
		Unmatched:              2,
		Duplicates:             4,
		BlankDuplicateArticles: 5,
		SourceItems:            120,
		SourceArticles:         100,
		SourceSheet:            "Отчет",
		SourceHeaderRow:        7,
		BlankSheet:             "Заказ",
		BlankHeaderRow:         3,
	}

	encoded, err := json.Marshal(summaryToDTO(summary))
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}
	assertKeys(t, object(t, encoded), []string{
		"brand", "order_month_label", "adjustment_label", "actual_main_period",
		"actual_previous_period", "source_city", "city_rule", "delivery_weeks",
		"filled", "left_blank", "suspicious", "unmatched", "duplicates",
		"blank_duplicate_articles", "source_items", "source_articles",
		"source_sheet", "source_header_row", "blank_sheet", "blank_header_row",
	})

	var decoded summaryDTO
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if got := summaryToDomain(decoded); !reflect.DeepEqual(got, summary) {
		t.Fatalf("summary round trip mismatch: got %+v want %+v", got, summary)
	}
}

func TestReportRowsRoundTripWithValues(t *testing.T) {
	sourceRow := 42
	orderedFact := 12.5
	recommended := 30.25
	rounded := 30
	baseRounded := 28
	inserted := 30.0

	rows := []job.ReportRow{{
		Key:                 "row-1",
		Status:              "filled",
		BlankID:             "blank-1",
		BlankLabel:          "Бланк",
		BlankRow:            10,
		BlankQuantityColumn: 6,
		BlankArticle:        "ART-1",
		BlankName:           "Товар",
		BlankUnit:           "шт",
		BlankBoxSize:        "12",
		SourceRow:           &sourceRow,
		SourceArticle:       "ART-1",
		SourceName:          "Товар источника",
		HasOrderedFact:      true,
		OrderedFact:         &orderedFact,
		SourceComment:       "комментарий",
		Stock:               "5",
		InTransit:           "2",
		Recommended:         &recommended,
		Rounded:             &rounded,
		BaseRounded:         &baseRounded,
		Inserted:            &inserted,
		AutoComment:         "auto",
		AdjustmentLabel:     "+10%",
		BoxAdjusted:         true,
		Duplicate:           true,
		DuplicateCandidates: []job.DuplicateCandidate{{
			SourceRow:     43,
			SourceArticle: "ART-1",
			SourceName:    "Дубликат",
			Recommended:   11.5,
			Rounded:       12,
			Stock:         "1",
			InTransit:     "0",
		}},
		Editable:   true,
		Similarity: 0.87,
	}}

	encoded, err := json.Marshal(reportRowsToDTO(rows))
	if err != nil {
		t.Fatalf("marshal report rows: %v", err)
	}
	rowObject := firstObject(t, encoded)
	assertKeys(t, rowObject, []string{
		"key", "status", "blank_id", "blank_label", "blank_row", "blank_quantity_col",
		"blank_article", "blank_name", "blank_unit", "blank_box_size", "source_row",
		"source_article", "source_name", "has_ordered_fact", "ordered_fact",
		"source_comment", "stock", "in_transit", "recommended", "rounded",
		"base_rounded", "inserted", "auto_comment", "adjustment_label",
		"box_adjusted", "duplicate", "duplicate_candidates", "editable", "similarity",
	})
	assertKeys(t, firstObject(t, rowObject["duplicate_candidates"]), []string{
		"source_row", "source_article", "source_name", "recommended", "rounded",
		"stock", "in_transit",
	})

	var decoded []reportRowDTO
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal report rows: %v", err)
	}
	if got := reportRowsToDomain(decoded); !reflect.DeepEqual(got, rows) {
		t.Fatalf("report rows round trip mismatch: got %+v want %+v", got, rows)
	}
}

func TestReportRowsRoundTripWithNilPointers(t *testing.T) {
	rows := []job.ReportRow{{
		Key:                 "row-empty",
		Status:              "unmatched",
		DuplicateCandidates: []job.DuplicateCandidate{},
	}}

	encoded, err := json.Marshal(reportRowsToDTO(rows))
	if err != nil {
		t.Fatalf("marshal report rows: %v", err)
	}
	rowObject := firstObject(t, encoded)
	for _, key := range []string{"source_row", "ordered_fact", "recommended", "rounded", "base_rounded", "inserted"} {
		if string(rowObject[key]) != "null" {
			t.Fatalf("field %s: got %s want null", key, rowObject[key])
		}
	}
	if string(rowObject["duplicate_candidates"]) != "[]" {
		t.Fatalf("duplicate_candidates: got %s want []", rowObject["duplicate_candidates"])
	}

	var decoded []reportRowDTO
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal report rows: %v", err)
	}
	got := reportRowsToDomain(decoded)
	if !reflect.DeepEqual(got, rows) {
		t.Fatalf("report rows round trip mismatch: got %+v want %+v", got, rows)
	}
	if got[0].SourceRow != nil || got[0].OrderedFact != nil || got[0].Recommended != nil ||
		got[0].Rounded != nil || got[0].BaseRounded != nil || got[0].Inserted != nil {
		t.Fatalf("nullable fields should stay nil, got %+v", got[0])
	}
}

func TestEmptyReportRowsMarshalAsEmptyArray(t *testing.T) {
	encoded, err := json.Marshal(reportRowsToDTO(nil))
	if err != nil {
		t.Fatalf("marshal nil report rows: %v", err)
	}
	if string(encoded) != "[]" {
		t.Fatalf("nil report rows: got %s want []", encoded)
	}
}

func TestUnmarshalJSONIgnoresEmptyColumn(t *testing.T) {
	var rows []reportRowDTO
	if err := unmarshalJSON(nil, &rows, "rows"); err != nil {
		t.Fatalf("unmarshal empty column: %v", err)
	}
	if rows != nil {
		t.Fatalf("expected untouched target, got %#v", rows)
	}
}

func object(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode object: %v", err)
	}
	return decoded
}

func firstObject(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var decoded []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode array: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatalf("expected at least one element in %s", raw)
	}
	return decoded[0]
}

func assertKeys(t *testing.T, decoded map[string]json.RawMessage, want []string) {
	t.Helper()
	got := make([]string, 0, len(decoded))
	for key := range decoded {
		got = append(got, key)
	}
	sort.Strings(got)
	expected := append([]string(nil), want...)
	sort.Strings(expected)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("json keys mismatch:\n got %v\nwant %v", got, expected)
	}
}
