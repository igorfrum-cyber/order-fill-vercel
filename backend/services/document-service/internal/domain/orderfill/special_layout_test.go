package orderfill

import (
	"context"
	"testing"
)

func layoutSource(rows ...[]string) [][]string {
	return append([][]string{{"Период: 01.08.2025 - 31.07.2026"}, {"Прошлый период: 01.08.2025 - 31.10.2025"}, {"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"}}, rows...)
}

func TestFillSothysWritesEveryVariantBlock(t *testing.T) {
	source := newFakeWorkbook("Заказ", layoutSource([]string{"110-15", "Cream 50 мл", "10", "1", "0"}, []string{"110-20", "Cream 100 мл", "8", "2", "0"}))
	blank := newFakeWorkbook("Бланк", [][]string{{"", "English", "Русский", "Артикул", "Объем", "Ед.", "Цена", "Заказ", "Артикул", "Объем", "Ед.", "Цена", "Заказ"}, {"", "Cream", "Крем", "110-15", "50", "мл", "100", "", "110-20", "100", "мл", "200", ""}})
	result, err := Fill(FillCommand{Source: source, Blank: blank, OrderMonth: "2026-09", Brand: "sothys", BlankID: "blank", Matcher: exactMatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := blank.Sheet("Бланк")
	for col, want := range map[int]string{8: "10", 13: "8"} {
		if got := sheet.Value(2, col); got != want {
			t.Errorf("column %d = %q want %q", col, got, want)
		}
	}
	for key, col := range map[string]int{"blank:2:8": 7, "blank:2:13": 12} {
		row := rowByKey(t, result, key)
		if len(row.BudgetPrices) != 1 || row.BudgetPrices[0].Column != col {
			t.Errorf("%s prices = %+v", key, row.BudgetPrices)
		}
	}
	if err := ApplyFinalEdits(FinalizeCommand{Source: source, Blank: blank, Rows: result.Rows, Edits: []ManualEdit{{Key: "blank:2:13", Value: "6", Comment: "правка"}}, Brand: "sothys"}); err != nil {
		t.Fatal(err)
	}
	if sheet.Value(2, 13) != "6" || sheet.Value(2, 8) != "10" {
		t.Fatal("variant edit wrote wrong column")
	}
}

func TestFillNovacutanCreatesOrderColumn(t *testing.T) {
	source := newFakeWorkbook("Заказ", layoutSource([]string{"N1", "NOVACUTAN SBIO", "7", "0", "0"}))
	blank := newFakeWorkbook("NOVACUTAN", [][]string{{"Описание", "Цена", "Заказ от"}, {"NOVACUTAN SBIO", "100", "100"}})
	result, err := Fill(FillCommand{Source: source, Blank: blank, OrderMonth: "2026-09", Brand: "novacutan", BlankID: "blank", Matcher: layoutNameMatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := blank.Sheet("NOVACUTAN")
	if sheet.Value(1, 4) != "Заказ" || sheet.Value(2, 4) != "7" {
		t.Fatalf("created column = %q / %q", sheet.Value(1, 4), sheet.Value(2, 4))
	}
	if rowByKey(t, result, "blank:2").Inserted == nil {
		t.Fatal("name match not filled")
	}
	if err := ApplyFinalEdits(FinalizeCommand{Source: source, Blank: blank, Rows: result.Rows, Edits: []ManualEdit{{Key: "blank:2", Value: "9", Comment: "правка"}}, Brand: "novacutan"}); err != nil {
		t.Fatal(err)
	}
	if sheet.Value(2, 4) != "9" {
		t.Fatal("final edit failed")
	}
}

type layoutNameMatcher struct{}

func (layoutNameMatcher) Match(_ context.Context, blank, source []MatchItem, _ MatchOptions) ([]MatchResult, error) {
	out := []MatchResult{}
	for _, item := range blank {
		out = append(out, MatchResult{BlankID: item.ID, SourceID: source[0].ID, Category: CategoryToOrder, Score: 1, Reasons: MatchReasons{Source: "name"}})
	}
	return out, nil
}

func TestSothysInheritsNamesOnContinuationRows(t *testing.T) {
	source := newFakeWorkbook("Заказ", layoutSource([]string{"110-15", "Cream 50 мл", "10", "0", "0"}, []string{"110-20", "Cream 100 мл", "8", "0", "0"}))
	blank := newFakeWorkbook("Бланк", [][]string{{"", "English", "Русский", "Артикул", "Объем", "Ед.", "Цена", "Заказ"}, {"", "Cream", "Крем", "110-15", "50", "мл", "100"}, {"", "", "", "110-20", "100", "мл", "200"}})
	result, err := Fill(FillCommand{Source: source, Blank: blank, OrderMonth: "2026-09", Brand: "sothys", BlankID: "blank", Matcher: exactMatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	row := rowByKey(t, result, "blank:3:8")
	if row.BlankName != "Cream / Крем" {
		t.Fatalf("continuation name = %q", row.BlankName)
	}
}

func TestFillNovacutanMasksUseSupplierBoxes(t *testing.T) {
	source := newFakeWorkbook("Заказ", layoutSource([]string{"N1", "EYE FILLER MASK NOVACUTAN", "7", "0", "0"}))
	blank := newFakeWorkbook("NOVACUTAN", [][]string{{"Описание", "Цена", "Заказ от", "Кол-во"}, {"EYE FILLER MASK NOVACUTAN", "100", "10"}})
	result, err := Fill(FillCommand{Source: source, Blank: blank, OrderMonth: "2026-09", Brand: "novacutan", BlankID: "blank", Matcher: layoutNameMatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := blank.Sheet("NOVACUTAN")
	if got := sheet.Value(2, 4); got != "2" {
		t.Fatalf("mask supplier boxes = %q want 2", got)
	}
	row := rowByKey(t, result, "blank:2")
	if row.Rounded == nil || *row.Rounded != 7 || row.BaseRounded == nil || *row.BaseRounded != 2 || row.AutoComment != "коробки по 5 шт." {
		t.Fatalf("mask report = %+v", row)
	}
}

func TestFillNovacutanNameWarningKeepsWarningStatus(t *testing.T) {
	source := newFakeWorkbook("Заказ", layoutSource([]string{"N1", "NOVACUTAN SBIO", "7", "0", "0"}))
	blank := newFakeWorkbook("NOVACUTAN", [][]string{{"Описание", "Цена", "Заказ от", "Кол-во"}, {"NOVACUTAN SBIO", "100", "100"}})
	result, err := Fill(FillCommand{Source: source, Blank: blank, OrderMonth: "2026-09", Brand: "novacutan", BlankID: "blank", Matcher: layoutWarningMatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	if row := rowByKey(t, result, "blank:2"); row.Status != StatusWarningNameDiffers {
		t.Fatalf("status = %q", row.Status)
	}
}

type layoutWarningMatcher struct{}

func (layoutWarningMatcher) Match(_ context.Context, blank, source []MatchItem, _ MatchOptions) ([]MatchResult, error) {
	return []MatchResult{{BlankID: blank[0].ID, SourceID: source[0].ID, Category: CategoryCheckNameOrVolume, Score: .8, Reasons: MatchReasons{Source: "name"}}}, nil
}
