package orderfill

import (
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

func TestApplyFinalEditsWritesCommentWhenQuantityStaysAtBaseline(t *testing.T) {
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
	if got := source.Value(*row.SourceRow, 6); got != "10" {
		t.Fatalf("source «Заказано по факту» = %q, want 10", got)
	}
	if got := source.Value(*row.SourceRow, 7); got != "подтвердили заказ" {
		t.Fatalf("source «Комментарий» = %q, want the reviewer comment", got)
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
