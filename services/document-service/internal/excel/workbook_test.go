package excel

import (
	"bytes"
	"testing"
)

func TestWorkbookRoundTripReadsAndWritesCells(t *testing.T) {
	workbook := NewWorkbook()
	sheet := workbook.AddSheet("Order")
	sheet.SetText("A1", "Article")
	sheet.SetNumber("B2", 12.5)

	var first bytes.Buffer
	if err := workbook.Save(&first); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := Load(bytes.NewReader(first.Bytes()), int64(first.Len()))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	loadedSheet, ok := loaded.Sheet("Order")
	if !ok {
		t.Fatal("expected Order sheet")
	}
	if got, ok := loadedSheet.Value("A1"); !ok || got != "Article" {
		t.Fatalf("expected A1 Article, got value=%q ok=%v", got, ok)
	}
	if got, ok := loadedSheet.Value("B2"); !ok || got != "12.5" {
		t.Fatalf("expected B2 12.5, got value=%q ok=%v", got, ok)
	}

	loadedSheet.SetNumber("B2", 18)
	var second bytes.Buffer
	if err := loaded.Save(&second); err != nil {
		t.Fatalf("second Save returned error: %v", err)
	}
	reloaded, err := Load(bytes.NewReader(second.Bytes()), int64(second.Len()))
	if err != nil {
		t.Fatalf("second Load returned error: %v", err)
	}
	reloadedSheet, ok := reloaded.Sheet("Order")
	if !ok {
		t.Fatal("expected reloaded Order sheet")
	}
	if got, ok := reloadedSheet.Value("B2"); !ok || got != "18" {
		t.Fatalf("expected updated B2 18, got value=%q ok=%v", got, ok)
	}
}
