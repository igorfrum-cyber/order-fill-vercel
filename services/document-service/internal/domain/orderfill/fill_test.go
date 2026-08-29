package orderfill

import (
	"errors"
	"testing"
)

func sourceGrid() [][]string {
	return [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"A100", "Крем для лица 50 мл", "10", "0", "0", "", ""},
		{"A200", "Гель для умывания 100 мл", "1.2", "4", "0", "", ""},
		{"A300", "Тоник 200 мл", "11", "1", "0", "", ""},
		{"A400", "Сыворотка 30 мл", "5", "0", "0", "", ""},
	}
}

func blankGrid() [][]string {
	return [][]string{
		{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
		{"A100", "Крем для лица", "50 мл", "", "3"},
		{"A200", "Гель для умывания", "100 мл", "", "3"},
		{"A300", "Тоник", "200 мл", "", "3"},
		{"A999", "Неизвестный товар", "10 мл", "", "3"},
	}
}

func fillFixture(t *testing.T) Result {
	t.Helper()
	result, err := Fill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	return result
}

func rowByKey(t *testing.T, result Result, key string) ReportRow {
	t.Helper()
	for _, row := range result.Rows {
		if row.Key == key {
			return row
		}
	}
	t.Fatalf("report row %q was not found", key)
	return ReportRow{}
}

func TestFillWritesRecommendedQuantityIntoBlank(t *testing.T) {
	result := fillFixture(t)
	sheet, _ := result.Blank.Sheet("Бланк")

	if got := sheet.Value(2, 4); got != "10" {
		t.Fatalf("expected quantity 10 in the blank, got %q", got)
	}
	row := rowByKey(t, result, "blank-1:2")
	if row.Status != StatusMatched {
		t.Fatalf("expected status %q, got %q", StatusMatched, row.Status)
	}
	if row.Inserted == nil || *row.Inserted != 10 {
		t.Fatalf("expected inserted 10, got %v", row.Inserted)
	}
}

func TestFillRoundsUpToBoxWhenCloseEnough(t *testing.T) {
	result := fillFixture(t)
	sheet, _ := result.Blank.Sheet("Бланк")

	if got := sheet.Value(4, 4); got != "12" {
		t.Fatalf("expected quantity rounded up to 12, got %q", got)
	}
	row := rowByKey(t, result, "blank-1:4")
	if !row.BoxAdjusted || row.AutoComment != "до коробки" {
		t.Fatalf("expected box adjustment with comment, got adjusted=%v comment=%q", row.BoxAdjusted, row.AutoComment)
	}
}

func TestFillLeavesNonPositiveRecommendationEmpty(t *testing.T) {
	result := fillFixture(t)
	sheet, _ := result.Blank.Sheet("Бланк")

	if got := sheet.Value(3, 4); got != "" {
		t.Fatalf("expected an empty quantity cell, got %q", got)
	}
	row := rowByKey(t, result, "blank-1:3")
	if row.Status != StatusLeftBlank {
		t.Fatalf("expected status %q, got %q", StatusLeftBlank, row.Status)
	}
	if row.Inserted != nil {
		t.Fatalf("expected no inserted quantity, got %v", *row.Inserted)
	}
}

func TestFillMarksBlankRowsMissingFromSource(t *testing.T) {
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:5")

	if row.Status != StatusNotInSource {
		t.Fatalf("expected status %q, got %q", StatusNotInSource, row.Status)
	}
	if row.Editable {
		t.Fatal("a row without a source position must not be editable")
	}
	if result.Summary.Unmatched != 1 {
		t.Fatalf("expected one unmatched row, got %d", result.Summary.Unmatched)
	}
}

func TestFillReportsSourceItemsMissingFromBlank(t *testing.T) {
	result := fillFixture(t)
	found := false
	for _, row := range result.Rows {
		if row.Status == StatusNotInBlank && row.SourceArticle == "A400" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the source position missing from the blank to be reported")
	}
}

func TestFillSummaryCountsPositions(t *testing.T) {
	result := fillFixture(t)
	if result.Summary.Filled != 2 {
		t.Fatalf("expected 2 filled rows, got %d", result.Summary.Filled)
	}
	if result.Summary.LeftBlank != 1 {
		t.Fatalf("expected 1 empty row, got %d", result.Summary.LeftBlank)
	}
	if result.Summary.Brand != "ANGIOPHARM" {
		t.Fatalf("unexpected brand %q", result.Summary.Brand)
	}
	if result.Summary.OrderMonthLabel != "сентябрь 2026" {
		t.Fatalf("unexpected order month label %q", result.Summary.OrderMonthLabel)
	}
}

func TestFillDoesNotRepeatSourceDuplicatesAlreadyFlaggedOnBlankRows(t *testing.T) {
	source := newFakeWorkbook("Заказ", [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"6014", "АН Косметичка непромокаемая", "0", "45", "0", "", ""},
		{"6014", "АН Косметичка матовая", "0", "10", "0", "", ""},
		{"ZZ99", "Только в таблице дважды", "4", "0", "0", "", ""},
		{"ZZ99", "Только в таблице дважды другой", "1", "0", "0", "", ""},
	})
	blank := newFakeWorkbook("Бланк", [][]string{
		{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
		{"6014", "Косметичка непромокаемая", "1", "", "3"},
	})
	result, err := Fill(FillCommand{
		Source:     source,
		Blank:      blank,
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}

	blankDuplicates := 0
	sourceOnlyDuplicates := 0
	for _, row := range result.Rows {
		if row.Status == StatusSourceDuplicate {
			sourceOnlyDuplicates++
			if row.BlankArticle == "6014" {
				t.Fatal("article 6014 is already flagged on the blank row and must not be repeated")
			}
		}
		if row.Duplicate && row.Status != StatusSourceDuplicate {
			blankDuplicates++
		}
	}
	if result.Summary.Duplicates != 1 {
		t.Fatalf("summary duplicates = %d, want 1 blank row", result.Summary.Duplicates)
	}
	if blankDuplicates != 1 {
		t.Fatalf("blank duplicate rows = %d, want 1", blankDuplicates)
	}
	if sourceOnlyDuplicates != 1 {
		t.Fatalf("source-only duplicate groups = %d, want 1 for ZZ99", sourceOnlyDuplicates)
	}
}

func TestFillRejectsExportBuiltForAnotherMonth(t *testing.T) {
	_, err := Fill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-10",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected an invalid input error, got %v", err)
	}
}

func TestFillRejectsUnsupportedBlankLayout(t *testing.T) {
	_, err := Fill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-09",
		Brand:      "novacutan",
		BlankID:    "blank-1",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected an invalid input error, got %v", err)
	}
}
