package north

import (
	"order-fill/backend/services/document-service/internal/adapter/outbound/xlsx"
	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
	"testing"
)

func TestNorthSelectedPriceUpdatesInvoiceAndPreservesBase(t *testing.T) {
	codec := xlsx.NewCodec()
	book, err := codec.(spreadsheet.TableCodec).NewTable("Сургут", []string{"Артикул", "Наименование", "Цена розничная", "Цена со скидкой", "Заказ", "Сумма"}, [][]any{{"A1", "Cream", 100, 70, 2, 140}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := book.Sheet("Сургут")
	sheet.(spreadsheet.FormulaWriter).SetFormula(2, 6, "D2*E2", 140)
	rule := brand.Rule("skin_synergy")
	needs, err := NeedsFromBlank(book, rule, "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(needs[0].Prices) != 2 {
		t.Fatalf("prices=%+v", needs[0].Prices)
	}
	report := BuildReport(rule.Key, needs, []Stock{{Article: "A1", Category: "A", MonthlyDemand: 10}}, []Planned{{Article: "A1", SupplierQty: 2}}, nil)
	if err := ApplyEdits(&report, []Edit{{Key: "A1", Value: "2", Comment: `{"cities":{"surgut":2},"price_column":3,"discount":10}`}}); err != nil {
		t.Fatal(err)
	}
	if report.PlanRows[0].BudgetPrice != 100 || report.PlanRows[0].BudgetPriceColumn != 3 {
		t.Fatalf("selected price=%+v", report.PlanRows[0])
	}
	if err := ApplySupplierPricing(book, rule, "", report.PlanRows); err != nil {
		t.Fatal(err)
	}
	if sheet.Value(2, 3) != "100" || sheet.Value(2, 4) != "90" || sheet.Value(2, 6) != "180" {
		t.Fatalf("base=%s invoice=%s total=%s", sheet.Value(2, 3), sheet.Value(2, 4), sheet.Value(2, 6))
	}
	raw, err := book.Save()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := codec.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := restored.Sheet("Сургут")
	if after.Value(2, 6) != "180" {
		t.Fatal("invoice cache not saved")
	}
	original, err := codec.(spreadsheet.TableCodec).NewTable("Сургут", []string{"Артикул", "Наименование", "Цена розничная", "Цена со скидкой", "Заказ", "Сумма"}, [][]any{{"A1", "Cream", 100, 70, 2, 140}})
	if err != nil {
		t.Fatal(err)
	}
	originalSheet, _ := original.Sheet("Сургут")
	originalSheet.(spreadsheet.FormulaWriter).SetFormula(2, 6, "D2*E2", 140)
	if err := ApplySupplierPricing(original, rule, "", report.PlanRows); err != nil {
		t.Fatal(err)
	}
	if originalSheet.Value(2, 4) != "90" {
		t.Fatal("repeat export compounded discount")
	}
}

func TestNorthSothysPricingKeepsIndependentVolumeBlocks(t *testing.T) {
	book, err := xlsx.NewCodec().(spreadsheet.TableCodec).NewTable("Сургут", []string{"", "English", "Русский", "Артикул", "Объем", "Ед.", "Цена", "Заказ", "Артикул", "Объем", "Ед.", "Цена", "Заказ"}, [][]any{{"", "Cream", "Крем", "110-15", "50", "мл", 100, 2, "110-20", "100", "мл", 200, 3}})
	if err != nil {
		t.Fatal(err)
	}
	rule := brand.Rule("sothys")
	needs, err := NeedsFromBlank(book, rule, "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(needs) != 2 || len(needs[0].Prices) != 1 || needs[0].Prices[0].Column != 7 || needs[1].Prices[0].Column != 12 {
		t.Fatalf("block prices=%+v", needs)
	}
	plans := []PlanRow{}
	for _, need := range needs {
		plans = append(plans, PlanRow{Key: need.Article, BudgetPrices: need.Prices, BudgetDefaultPriceColumn: need.PriceColumn, BudgetDiscount: 10, ActualSupplierOrder: need.Qty})
	}
	if err := ApplySupplierPricing(book, rule, "", plans); err != nil {
		t.Fatal(err)
	}
	sheet, _ := book.Sheet("Сургут")
	if sheet.Value(2, 7) != "90" || sheet.Value(2, 12) != "180" {
		t.Fatalf("block prices=%s/%s", sheet.Value(2, 7), sheet.Value(2, 12))
	}
}

func TestNorthPriceSelectionMapsByHeaderLabel(t *testing.T) {
	book, err := xlsx.NewCodec().(spreadsheet.TableCodec).NewTable("Сургут", []string{"Артикул", "Наименование", "Цена розничная", "Цена со скидкой", "Заказ", "Сумма"}, [][]any{{"A1", "Cream", 100, 70, 2, 140}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := book.Sheet("Сургут")
	sheet.(spreadsheet.FormulaWriter).SetFormula(2, 6, "D2*E2", 140)
	plans := []PlanRow{{Key: "A1", ActualSupplierOrder: 2, BudgetDiscount: 10, BudgetPriceColumn: 9, BudgetPrices: []orderfill.BudgetPrice{{Column: 9, Label: "Цена розничная", Price: 100}}}}
	if err := ApplySupplierPricing(book, brand.Rule("skin_synergy"), "", plans); err != nil {
		t.Fatal(err)
	}
	if sheet.Value(2, 4) != "90" {
		t.Fatalf("invoice=%s", sheet.Value(2, 4))
	}
}
