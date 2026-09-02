package jobstore

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"order-fill/services/document-service/internal/app/port"
	"order-fill/services/document-service/internal/domain/orderfill"
)

func TestOutputFilesRoundTrip(t *testing.T) {
	domain := []port.OutputFile{{
		ID:          "jobs/job-1/outputs/blank.xlsx",
		Label:       "Скачать заполненный бланк",
		Name:        "blank.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		SizeBytes:   2048,
		StorageKey:  "jobs/job-1/outputs/blank.xlsx",
	}}

	encoded, err := marshalJSON(outputFilesToDTO(domain), "output_files")
	if err != nil {
		t.Fatalf("marshal output files: %v", err)
	}

	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decode output files: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("output files: got %d want 1", len(raw))
	}
	assertKeys(t, raw[0], []string{"id", "label", "name", "content_type", "size_bytes", "storage_key"})

	var decoded []outputFileDTO
	if err := unmarshalJSON(encoded, &decoded, "output_files"); err != nil {
		t.Fatalf("unmarshal output files: %v", err)
	}
	if got := outputFilesToDomain(decoded); !reflect.DeepEqual(got, domain) {
		t.Fatalf("output files round trip: got %+v want %+v", got, domain)
	}
}

func TestEmptyOutputFilesMarshalAsArray(t *testing.T) {
	for name, files := range map[string][]port.OutputFile{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			encoded, err := marshalJSON(outputFilesToDTO(files), "output_files")
			if err != nil {
				t.Fatalf("marshal output files: %v", err)
			}
			if string(encoded) != "[]" {
				t.Fatalf("output files: got %s want []", encoded)
			}
		})
	}
}

func TestSummaryRoundTrip(t *testing.T) {
	domain := orderfill.Summary{
		Brand:                  "ANGIOPHARM",
		OrderMonthLabel:        "сентябрь 2026",
		AdjustmentLabel:        "коробки",
		ActualMainPeriod:       "август 2026",
		ActualPreviousPeriod:   "июль 2026",
		SourceCity:             "Новый Уренгой",
		CityRule:               "Новый Уренгой",
		DeliveryWeeks:          2.5,
		Filled:                 12,
		LeftBlank:              3,
		Suspicious:             1,
		Unmatched:              4,
		Duplicates:             2,
		BlankDuplicateArticles: 5,
		SourceItems:            120,
		SourceArticles:         118,
		SourceSheet:            "Расчет заказа",
		SourceHeaderRow:        7,
		BlankSheet:             "Бланк",
		BlankHeaderRow:         3,
	}

	encoded, err := marshalJSON(summaryToDTO(domain), "summary")
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	assertKeys(t, raw, []string{
		"brand", "order_month_label", "adjustment_label", "actual_main_period",
		"actual_previous_period", "source_city", "city_rule", "delivery_weeks",
		"filled", "left_blank", "suspicious", "unmatched", "duplicates",
		"not_in_blank", "blank_duplicate_articles", "source_items", "source_articles",
		"source_sheet", "source_header_row", "blank_sheet", "blank_header_row",
	})

	var decoded summaryDTO
	if err := unmarshalJSON(encoded, &decoded, "summary"); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if got := summaryToDomain(decoded); !reflect.DeepEqual(got, domain) {
		t.Fatalf("summary round trip: got %+v want %+v", got, domain)
	}
}

func TestReportRowRoundTrip(t *testing.T) {
	sourceRow := 42
	orderedFact := 7.5
	recommended := 9.25
	rounded := 9
	baseRounded := 10
	inserted := 12.0

	domain := []orderfill.ReportRow{{
		Key:                 "blank-1:row:12",
		Status:              "filled",
		BlankID:             "blank-1",
		BlankLabel:          "blank.xlsx",
		BlankRow:            12,
		BlankQuantityColumn: 5,
		BlankArticle:        "AP-100",
		BlankName:           "Ангиофарм",
		BlankUnit:           "шт",
		BlankBoxSize:        "10",
		SourceRow:           &sourceRow,
		SourceArticle:       "100",
		SourceName:          "Ангиофарм 30 таб",
		HasOrderedFact:      true,
		OrderedFact:         &orderedFact,
		SourceComment:       "остаток",
		Stock:               "15",
		InTransit:           "5",
		Recommended:         &recommended,
		Rounded:             &rounded,
		BaseRounded:         &baseRounded,
		Inserted:            &inserted,
		AutoComment:         "округлено до коробки",
		AdjustmentLabel:     "коробки",
		BoxAdjusted:         true,
		Duplicate:           true,
		DuplicateCandidates: []orderfill.DuplicateCandidate{{
			SourceRow:     42,
			SourceArticle: "100",
			SourceName:    "Ангиофарм 30 таб",
			Recommended:   9.25,
			Rounded:       9,
			Stock:         "15",
			InTransit:     "5",
		}},
		Editable:   true,
		Similarity: 0.9876,
	}}

	encoded, err := marshalJSON(reportRowsToDTO(domain), "rows")
	if err != nil {
		t.Fatalf("marshal rows: %v", err)
	}

	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("rows: got %d want 1", len(raw))
	}
	assertKeys(t, raw[0], reportRowKeys())

	var candidates []map[string]json.RawMessage
	if err := json.Unmarshal(raw[0]["duplicate_candidates"], &candidates); err != nil {
		t.Fatalf("decode duplicate candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("duplicate candidates: got %d want 1", len(candidates))
	}
	assertKeys(t, candidates[0], []string{
		"source_row", "source_article", "source_name", "recommended", "rounded", "stock", "in_transit",
	})

	var decoded []reportRowDTO
	if err := unmarshalJSON(encoded, &decoded, "rows"); err != nil {
		t.Fatalf("unmarshal rows: %v", err)
	}
	if got := reportRowsToDomain(decoded); !reflect.DeepEqual(got, domain) {
		t.Fatalf("rows round trip: got %+v want %+v", got[0], domain[0])
	}
}

// The domain leaves the optional quantities nil for rows the engine could not
// compute; api-service relies on those arriving as JSON null.
func TestReportRowNilPointersRoundTrip(t *testing.T) {
	domain := []orderfill.ReportRow{{
		Key:                 "blank-1:row:13",
		Status:              "not_in_source",
		BlankID:             "blank-1",
		BlankQuantityColumn: 5,
		DuplicateCandidates: nil,
	}}

	encoded, err := marshalJSON(reportRowsToDTO(domain), "rows")
	if err != nil {
		t.Fatalf("marshal rows: %v", err)
	}

	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	assertKeys(t, raw[0], reportRowKeys())

	for _, key := range []string{"source_row", "ordered_fact", "recommended", "rounded", "base_rounded", "inserted"} {
		if got := string(raw[0][key]); got != "null" {
			t.Fatalf("%s: got %s want null", key, got)
		}
	}
	if got := string(raw[0]["duplicate_candidates"]); got != "[]" {
		t.Fatalf("duplicate_candidates: got %s want []", got)
	}

	var decoded []reportRowDTO
	if err := unmarshalJSON(encoded, &decoded, "rows"); err != nil {
		t.Fatalf("unmarshal rows: %v", err)
	}
	rows := reportRowsToDomain(decoded)
	if len(rows) != 1 {
		t.Fatalf("rows: got %d want 1", len(rows))
	}

	row := rows[0]
	if row.SourceRow != nil || row.OrderedFact != nil || row.Recommended != nil ||
		row.Rounded != nil || row.BaseRounded != nil || row.Inserted != nil {
		t.Fatalf("optional quantities must round trip back to nil: %+v", row)
	}
	if row.DuplicateCandidates == nil || len(row.DuplicateCandidates) != 0 {
		t.Fatalf("duplicate candidates: got %+v want an empty slice", row.DuplicateCandidates)
	}

	want := domain[0]
	want.DuplicateCandidates = []orderfill.DuplicateCandidate{}
	if !reflect.DeepEqual(row, want) {
		t.Fatalf("rows round trip: got %+v want %+v", row, want)
	}
}

func TestEmptyRowsMarshalAsArray(t *testing.T) {
	for name, rows := range map[string][]orderfill.ReportRow{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			encoded, err := marshalJSON(reportRowsToDTO(rows), "rows")
			if err != nil {
				t.Fatalf("marshal rows: %v", err)
			}
			if string(encoded) != "[]" {
				t.Fatalf("rows: got %s want []", encoded)
			}
		})
	}
}

func TestDuplicateCandidatesRoundTrip(t *testing.T) {
	domain := []orderfill.DuplicateCandidate{{
		SourceRow:     11,
		SourceArticle: "AP-200",
		SourceName:    "Дубликат",
		Recommended:   3.5,
		Rounded:       4,
		Stock:         "1",
		InTransit:     "2",
	}}

	encoded, err := marshalJSON(duplicateCandidatesToDTO(domain), "duplicate_candidates")
	if err != nil {
		t.Fatalf("marshal duplicate candidates: %v", err)
	}

	var decoded []duplicateCandidateDTO
	if err := unmarshalJSON(encoded, &decoded, "duplicate_candidates"); err != nil {
		t.Fatalf("unmarshal duplicate candidates: %v", err)
	}
	if got := duplicateCandidatesToDomain(decoded); !reflect.DeepEqual(got, domain) {
		t.Fatalf("duplicate candidates round trip: got %+v want %+v", got, domain)
	}
}

func TestUnmarshalJSONIgnoresEmptyColumn(t *testing.T) {
	var rows []reportRowDTO
	if err := unmarshalJSON(nil, &rows, "rows"); err != nil {
		t.Fatalf("unmarshal empty column: %v", err)
	}
	if rows != nil {
		t.Fatalf("rows: got %+v want nil", rows)
	}
}

func reportRowKeys() []string {
	return []string{
		"key", "status", "blank_id", "blank_label", "blank_row", "blank_quantity_col",
		"blank_article", "blank_name", "blank_unit", "blank_box_size", "source_row",
		"source_article", "source_name", "has_ordered_fact", "ordered_fact",
		"source_comment", "stock", "in_transit", "recommended", "rounded",
		"base_rounded", "inserted", "auto_comment", "adjustment_label", "box_adjusted",
		"duplicate", "duplicate_candidates", "editable", "similarity",
	}
}

func assertKeys(t *testing.T, actual map[string]json.RawMessage, want []string) {
	t.Helper()

	got := make([]string, 0, len(actual))
	for key := range actual {
		got = append(got, key)
	}
	sort.Strings(got)

	expected := append([]string(nil), want...)
	sort.Strings(expected)

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("json keys:\n got %v\nwant %v", got, expected)
	}
}
