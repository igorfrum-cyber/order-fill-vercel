package orderfill

import (
	"errors"
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
	if got := sheet.Value(7, 1); got != "A3" {
		t.Fatalf("warehouse-only article = %q, want A3", got)
	}
	if got := sheet.Value(8, 1); got != "A4" {
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

	merged, err := MergeTyumenSources(office, warehouse, "2026-09", brand.Rule("angiopharm"))
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
