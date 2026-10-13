package orderfill

import (
	"context"
	"errors"
	"strings"
	"testing"

	"order-fill/backend/services/document-service/internal/domain/brand"
)

func tyumenLocationGrid(month string, rows ...[]string) [][]string {
	grid := [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"", "", month, "Итого"},
		{"Артикул", "Товар", "Количество", "Количество", "Сумма выручки", "% выручки", "Кумулятивный %", "Категория", "Среднее в месяц", "Количество прошлый период", "Целевой запас", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
	}
	return append(grid, rows...)
}

func TestMergeTyumenSourcesSumsLocationsAndKeepsUniqueRows(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025",
		[]string{"A1", "Крем", "10", "10", "100", "", "", "A", "", "2", "", "", "2", "1"},
		[]string{"A2", "Гель", "4", "4", "40", "", "", "B", "", "1", "", "", "1", "0"},
	))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025",
		[]string{"A1", "Крем", "5", "5", "50", "", "", "A", "", "3", "", "", "8", "2"},
		[]string{"A3", "Маска", "7", "7", "70", "", "", "C", "", "1", "", "", "3", "0"},
		[]string{"A4", "Тоник", "2", "2", "20", "", "", "C", "", "1", "", "", "4", "0"},
	))

	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatalf("merge Tyumen locations: %v", err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if got := sheet.Value(5, 3); got != "15" {
		t.Fatalf("combined sales = %q, want 15", got)
	}
	if got := sheet.Value(5, 13); got != "10" {
		t.Fatalf("combined stock = %q, want 10", got)
	}
	if got := sheet.Value(5, 14); got != "3" {
		t.Fatalf("combined in-transit = %q, want 3", got)
	}
	if got := sheet.Value(6, 1); got != "A3" {
		t.Fatalf("warehouse-only article = %q, want A3", got)
	}
	if got := sheet.Value(7, 1); got != "A4" {
		t.Fatalf("second warehouse-only article = %q, want A4", got)
	}
}

func TestMergeTyumenSourcesRejectsDifferentSalesMonths(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "1", "1", "10", "", "", "A", "", "1", "", "", "1", "0"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Июль 2025", []string{"A1", "Крем", "1", "1", "10", "", "", "A", "", "1", "", "", "1", "0"}))

	_, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestMergeTyumenSourcesMatchesByNameWhenOneArticleIsMissingAndKeepsFacts(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025",
		[]string{"", "Крем", "10", "10", "100", "", "", "A", "", "2", "", "", "2", "1", "4", "офис"},
	))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025",
		[]string{"A1", "Крем", "5", "5", "50", "", "", "A", "", "3", "", "", "8", "2", "3", "склад"},
	))

	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"), TyumenMergeOptions{PreserveFacts: true})
	if err != nil {
		t.Fatalf("merge Tyumen locations: %v", err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if got := sheet.Bounds().MaxRow; got != 5 {
		t.Fatalf("rows after name match = %d, want 5", got)
	}
	if got := sheet.Value(5, 1); got != "A1" {
		t.Fatalf("merged article = %q, want A1", got)
	}
	if got := sheet.Value(5, 15); got != "7" {
		t.Fatalf("combined ordered fact = %q, want 7", got)
	}
	if got := sheet.Value(5, 16); got != "офис; склад" {
		t.Fatalf("combined comment = %q, want office and warehouse comments", got)
	}
}

func TestMergeTyumenSourcesAcceptsWarehouseWithoutSalesHistory(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025",
		[]string{"A1", "Крем", "10", "10", "100", "", "", "A", "", "2", "", "", "2", "1"},
	))
	warehouse := newFakeWorkbook("Заказ", [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"A1", "Крем", "", "8", "2", "", ""},
	})

	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatalf("merge warehouse without sales: %v", err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if got := sheet.Value(5, 13); got != "10" {
		t.Fatalf("combined stock = %q, want 10", got)
	}
	if got := sheet.Value(5, 14); got != "3" {
		t.Fatalf("combined transit = %q, want 3", got)
	}
}

func TestMergeTyumenSourcesPreservesConflictsForReview(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "10", "10", "100", "", "", "A", "", "2", "", "", "2", "1"}, []string{"A1", "Гель", "4", "4", "40", "", "", "B", "", "1", "", "", "1", "0"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "5", "5", "50", "", "", "A", "", "3", "", "", "8", "2"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if sheet.Bounds().MaxRow != 7 {
		t.Fatalf("rows = %d, want three independent products", sheet.Bounds().MaxRow)
	}
	for row := 5; row <= 7; row++ {
		if !strings.Contains(sheet.Value(row, 16), "Проверить:") {
			t.Errorf("row %d comment = %q", row, sheet.Value(row, 16))
		}
	}
}

func TestMergeTyumenSourcesUsesWarehouseTemplateAndLocationColumns(t *testing.T) {
	office := newFakeWorkbook("Офис", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "10", "10", "100", "", "", "A", "", "2", "", "", "2", "1", "4", "офис"}))
	warehouse := newFakeWorkbook("Склад", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "5", "5", "50", "", "", "A", "", "3", "", "", "8", "2", "3", "склад"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, ok := merged.Sheet("Склад")
	if !ok {
		t.Fatal("warehouse template not retained")
	}
	for col, want := range map[int]string{17: "2", 18: "8", 19: "1", 20: "2"} {
		if got := sheet.Value(5, col); got != want {
			t.Errorf("col %d = %q want %q", col, got, want)
		}
	}
	if got := sheet.Value(5, 15); got != "" {
		t.Errorf("default fact = %q, want cleared", got)
	}
}

func TestMergeTyumenSourcesAcceptsInventoryWithoutPeriods(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "10", "10", "100", "", "", "A", "", "2", "", "", "2", "1"}))
	warehouse := newFakeWorkbook("Заказ", [][]string{{"Срок поставки: 5 недель"}, {"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"}, {"A1", "Крем", "", "8", "2"}})
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if sheet.Value(5, 13) != "10" {
		t.Fatal("stock not combined")
	}
}

func TestMergeTyumenSourcesCanonicalizesLeviSsimeIdentity(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"123", "Лев Крем отбеливающая 50 мл / Lightening White Face Cream", "10", "10", "100", "", "", "A", "", "2", "", "", "2", "1"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"MT123", "LeviSsime Крем осветляющая 50 мл / Lightening White Face Cream", "5", "5", "50", "", "", "A", "", "3", "", "", "8", "2"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("levissime"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if sheet.Bounds().MaxRow != 5 || sheet.Value(5, 13) != "10" {
		t.Fatalf("identity not merged: rows %d stock %q", sheet.Bounds().MaxRow, sheet.Value(5, 13))
	}
}

func TestMergeTyumenSourcesKeepsSameArticleDifferentNamesSeparate(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "1", "1", "10", "", "", "A", "", "1", "", "", "2", "0"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Гель", "3", "3", "30", "", "", "A", "", "1", "", "", "8", "0"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if sheet.Bounds().MaxRow != 6 || sheet.Value(5, 13) != "8" || sheet.Value(6, 13) != "2" {
		t.Fatal("conflicting stock must remain separate")
	}
}

func TestMergeTyumenSourcesRejectsWarehouseUnallocatedRevenue(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "1", "1", "10", "", "", "A", "", "1", "", "", "2", "0"}))
	warehouse := newFakeWorkbook("Заказ", [][]string{{"Артикул", "Товар", "Сумма выручки", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"}, {"A1", "Крем", "10", "", "8", "0"}})
	if _, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm")); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want missing monthly allocation", err)
	}
}

func TestMergeTyumenSourcesMergesChzBeforeMatchingLocations(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "1", "1", "10", "", "", "A", "", "1", "", "", "2", "0"}, []string{"A1", "ЧЗ Крем", "3", "3", "30", "", "", "A", "", "1", "", "", "4", "0"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "5", "5", "50", "", "", "A", "", "1", "", "", "8", "0"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"), TyumenMergeOptions{Chz: warehouseChzMerger{}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if sheet.Bounds().MaxRow != 5 || sheet.Value(5, 13) != "14" || sheet.Value(5, 2) != "ЧЗ + Крем" {
		t.Fatalf("CHЗ merge = rows %d stock %q name %q", sheet.Bounds().MaxRow, sheet.Value(5, 13), sheet.Value(5, 2))
	}
}

func TestMergeTyumenSourcesUsesNovacutanProductKeys(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"", "Сыворотка NOVACUTAN Bio Pro", "1", "1", "10", "", "", "A", "", "1", "", "", "2", "0"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"", "NOVACUTAN Biopro сыворотка", "3", "3", "30", "", "", "A", "", "1", "", "", "8", "0"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("novacutan"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if sheet.Bounds().MaxRow != 6 || !strings.Contains(sheet.Value(5, 16), "Проверить:") {
		t.Fatal("main flags incompatible names with same Novacutan product identity")
	}
}

type warehouseChzMerger struct{}

func (warehouseChzMerger) MergeChz(_ context.Context, items []MatchItem, _ MatchOptions) ([]ChzMerge, error) {
	if len(items) == 2 {
		return []ChzMerge{{TargetID: items[0].ID, CloneIDs: []string{items[1].ID}}}, nil
	}
	return nil, nil
}

func TestMergeTyumenSourcesNormalDuplicateMergeKeepsNormalName(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "1", "1", "10", "", "", "A", "", "1", "", "", "2", "0"}, []string{"A1", "Крем", "3", "3", "30", "", "", "A", "", "1", "", "", "4", "0"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "5", "5", "50", "", "", "A", "", "1", "", "", "8", "0"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("levissime"), TyumenMergeOptions{Chz: warehouseChzMerger{}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if sheet.Value(5, 2) != "Крем" {
		t.Fatalf("normal name = %q", sheet.Value(5, 2))
	}
	officeSheet, _ := office.Sheet("Заказ")
	if officeSheet.Value(5, 2) != "Крем" {
		t.Fatalf("normal duplicate name = %q", officeSheet.Value(5, 2))
	}
}

func TestMergeTyumenSourcesRefreshesTotalsAfterRecommendations(t *testing.T) {
	office := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "1", "1", "10", "", "", "A", "", "1", "", "", "2", "0"}, []string{"A2", "Гель", "3", "3", "30", "", "", "A", "", "1", "", "", "4", "0"}))
	warehouse := newFakeWorkbook("Заказ", tyumenLocationGrid("Август 2025", []string{"A1", "Крем", "5", "5", "50", "", "", "A", "", "1", "", "", "8", "0"}, []string{"Итого"}))
	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatal(err)
	}
	detection, err := DetectSourceColumns(merged)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyRecommendations(FillCommand{Recommender: warehouseRecommendations{}}, detection, brand.Rule("angiopharm"), *detectCalculationColumns(detection), 1, ""); err != nil {
		t.Fatal(err)
	}
	sheet, _ := merged.Sheet("Заказ")
	if got := sheet.Value(6, 11); got != "40" {
		t.Fatalf("total target = %q want 40", got)
	}
	if got := sheet.Value(6, 12); got != "20" {
		t.Fatalf("total recommended = %q want 20", got)
	}
}

type warehouseRecommendations struct{}

func (warehouseRecommendations) Recommend(_ context.Context, _ string, _ string, _ float64, rows []RecommendRow) ([]RecommendRow, error) {
	for i := range rows {
		rows[i].TargetStock = 20
		rows[i].Recommended = 10
	}
	return rows, nil
}
