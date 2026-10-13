package orderfill

import (
	"order-fill/backend/services/document-service/internal/domain/brand"
	"testing"

	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

func TestApplyFinalEditsWritesFactAndCommentIntoSource(t *testing.T) {
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:2")
	if row.SourceRow == nil {
		t.Fatal("matched row must point at the 1C source line")
	}

	if err := ApplyFinalEdits(FinalizeCommand{
		Source: result.Source,
		Blank:  result.Blank,
		Rows:   result.Rows,
		Brand:  "angiopharm",
		Edits:  []ManualEdit{{Key: row.Key, Value: "18", Comment: "договорились с поставщиком"}},
	}); err != nil {
		t.Fatalf("apply edits: %v", err)
	}

	source := mustSheetByName(t, result.Source, "Заказ")
	if got := source.Value(*row.SourceRow, 6); got != "18" {
		t.Fatalf("source «Заказано по факту» = %q, want 18", got)
	}
	if got := source.Value(*row.SourceRow, 7); got != "договорились с поставщиком" {
		t.Fatalf("source «Комментарий» = %q, want the reviewer comment", got)
	}
	blank := mustSheetByName(t, result.Blank, "Бланк")
	if got := blank.Value(row.BlankRow, 4); got != "18" {
		t.Fatalf("blank quantity = %q, want 18", got)
	}
}

func TestApplyFinalEditsCanAddReliableStocksToBlank(t *testing.T) {
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:2")
	if row.SourceRow == nil {
		t.Fatal("matched row must point at the 1C source line")
	}
	if err := ApplyFinalEdits(FinalizeCommand{
		Source:             result.Source,
		Blank:              result.Blank,
		Rows:               result.Rows,
		Brand:              "angiopharm",
		IncludeBlankStocks: true,
	}); err != nil {
		t.Fatalf("apply edits with stock: %v", err)
	}

	blank := mustSheetByName(t, result.Blank, "Бланк")
	if got := blank.Value(1, 10); got != "Остаток" {
		t.Fatalf("stock header = %q, want Остаток", got)
	}
	if got := blank.Value(row.BlankRow, 10); got != "0" {
		t.Fatalf("matched stock = %q, want 0", got)
	}
	if got := blank.Value(5, 10); got != "" {
		t.Fatalf("unmatched stock = %q, want empty", got)
	}
}

func TestApplyFinalEditsClearsFactAndCommentWhenQuantityStaysAtBaseline(t *testing.T) {
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:2")
	if row.SourceRow == nil {
		t.Fatal("matched row must point at the 1C source line")
	}

	if err := ApplyFinalEdits(FinalizeCommand{
		Source: result.Source,
		Blank:  result.Blank,
		Rows:   result.Rows,
		Brand:  "angiopharm",
		Edits:  []ManualEdit{{Key: row.Key, Value: "10", Comment: "подтвердили заказ"}},
	}); err != nil {
		t.Fatalf("apply edits: %v", err)
	}

	source := mustSheetByName(t, result.Source, "Заказ")
	if got := source.Value(*row.SourceRow, 6); got != "" {
		t.Fatalf("source «Заказано по факту» = %q, want empty baseline fact", got)
	}
	if got := source.Value(*row.SourceRow, 7); got != "" {
		t.Fatalf("source «Комментарий» = %q, want no baseline comment", got)
	}
}

func TestApplyFinalEditsClearsFactWhenQuantityReturnsToBaselineWithoutComment(t *testing.T) {
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:2")
	if row.SourceRow == nil {
		t.Fatal("matched row must point at the 1C source line")
	}
	source := mustSheetByName(t, result.Source, "Заказ")
	source.SetNumber(*row.SourceRow, 6, 18)
	source.SetText(*row.SourceRow, 7, "старый комментарий")

	if err := ApplyFinalEdits(FinalizeCommand{
		Source: result.Source,
		Blank:  result.Blank,
		Rows:   result.Rows,
		Brand:  "angiopharm",
		Edits:  []ManualEdit{{Key: row.Key, Value: "10", Comment: ""}},
	}); err != nil {
		t.Fatalf("apply edits: %v", err)
	}

	if got := source.Value(*row.SourceRow, 6); got != "" {
		t.Fatalf("source «Заказано по факту» = %q, want empty", got)
	}
	if got := source.Value(*row.SourceRow, 7); got != "" {
		t.Fatalf("source «Комментарий» = %q, want empty", got)
	}
}

func mustSheetByName(t *testing.T, book spreadsheet.Workbook, name string) spreadsheet.Sheet {
	t.Helper()
	sheet, ok := book.Sheet(name)
	if !ok {
		t.Fatalf("sheet %q was not found", name)
	}
	return sheet
}

func TestApplyFinalEditsRejectsStaleAutoComment(t *testing.T) {
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:2")
	row.AutoComment = "до коробки"
	row.Inserted = new(10.0)
	err := ApplyFinalEdits(FinalizeCommand{Source: result.Source, Blank: result.Blank, Rows: []ReportRow{row}, Brand: "angiopharm", Edits: []ManualEdit{{Key: row.Key, Value: "18", Comment: "До коробки"}}})
	if err == nil {
		t.Fatal("main requires justification after changing auto-rounded quantity")
	}
}

func TestBlankStocksRejectTextualQuantitiesLikeMain(t *testing.T) {
	t.Parallel()
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:2")
	row.Stock = "5 шт"
	if err := AddOrderBlankStocks(result.Blank, []ReportRow{row}, brand.Rule("angiopharm"), "angiopharm"); err != nil {
		t.Fatal(err)
	}
	sheet := mustSheetByName(t, result.Blank, "Бланк")
	if sheet.Value(row.BlankRow, 10) != "" {
		t.Fatal("main leaves a textual stock empty")
	}
}
