package orderfill

import (
	"order-fill/backend/services/document-service/internal/adapter/outbound/xlsx"
	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
	"strconv"
	"testing"
)

func TestApplyBudgetPricingFallbackAndFormulaTotals(t *testing.T) {
	codec := xlsx.NewCodec()
	workbook, err := codec.(spreadsheet.TableCodec).NewTable("Blank", []string{"Артикул", "Цена", "Количество", "Сумма"}, [][]any{{"A", 100, 2, 200}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := workbook.Sheet("Blank")
	writer := sheet.(spreadsheet.FormulaWriter)
	writer.SetFormula(2, 4, "B2*C2", 200)
	rows := []ReportRow{{Key: "main:2", BlankRow: 2, BlankQuantityColumn: 3, BlankName: "A", Inserted: new(float64(2))}}
	entries := []BudgetPricing{{RowKey: "main:2", PriceColumn: 2, BasePrice: 100, Discount: 30, Price: 70}}
	if err := ApplyBudgetPricing(sheet, 1, "skin_synergy", rows, entries); err != nil {
		t.Fatal(err)
	}
	if sheet.Value(2, 2) != "70" || sheet.Value(2, 4) != "140" {
		t.Fatalf("price=%s total=%s", sheet.Value(2, 2), sheet.Value(2, 4))
	}
	raw, err := workbook.Save()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := codec.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	result, _ := restored.Sheet("Blank")
	if result.Value(2, 4) != "140" {
		t.Fatal("formula cache was not saved")
	}
}

func TestApplyBudgetPricingSharedDiscountAndChristina(t *testing.T) {
	for _, brand := range []string{"skin_synergy", "christina"} {
		t.Run(brand, func(t *testing.T) {
			workbook, err := xlsx.NewCodec().(spreadsheet.TableCodec).NewTable("Blank", []string{"", "", "", ""}, [][]any{{"", "", "", ""}, {"Артикул", "Цена", "Цена со скидкой", "Количество", "Сумма"}, {"A", 100, 100, 2, 200}})
			if err != nil {
				t.Fatal(err)
			}
			sheet, _ := workbook.Sheet("Blank")
			sheet.SetNumber(1, 2, 0)
			writer := sheet.(spreadsheet.FormulaWriter)
			writer.SetFormula(4, 3, "ROUND(B4*(1-$B$1/100),2)", 100)
			writer.SetFormula(4, 5, "C4*D4", 200)
			rows := []ReportRow{{Key: "main:4", BlankRow: 4, BlankQuantityColumn: 4, Inserted: new(float64(2))}}
			if err := ApplyBudgetPricing(sheet, 3, brand, rows, []BudgetPricing{{RowKey: "main:4", PriceColumn: 2, BasePrice: 100, Discount: 30, Price: 70}}); err != nil {
				t.Fatal(err)
			}
			expected := "30"
			price := "70"
			if brand == "christina" {
				expected = "0"
				price = "100"
			}
			if sheet.Value(1, 2) != expected || sheet.Value(4, 3) != price {
				t.Fatalf("discount=%s price=%s", sheet.Value(1, 2), sheet.Value(4, 3))
			}
		})
	}
}

func TestApplyBudgetPricingRejectsConflictingSharedDiscounts(t *testing.T) {
	workbook, err := xlsx.NewCodec().(spreadsheet.TableCodec).NewTable("Blank", []string{"", "", "", ""}, [][]any{{"", "", "", ""}, {"Артикул", "Цена", "Цена со скидкой", "Количество", "Сумма"}, {"A", 100, 100, 2, 200}, {"B", 200, 200, 2, 400}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := workbook.Sheet("Blank")
	sheet.SetNumber(1, 2, 0)
	writer := sheet.(spreadsheet.FormulaWriter)
	for row := 4; row <= 5; row++ {
		writer.SetFormula(row, 3, "B"+strconv.Itoa(row)+"*(1-$B$1/100)", 100)
		writer.SetFormula(row, 5, "C"+strconv.Itoa(row)+"*D"+strconv.Itoa(row), 200)
	}
	rows := []ReportRow{{Key: "a", BlankRow: 4, BlankQuantityColumn: 4}, {Key: "b", BlankRow: 5, BlankQuantityColumn: 4}}
	entries := []BudgetPricing{{RowKey: "a", PriceColumn: 2, BasePrice: 100, Discount: 30, Price: 70}, {RowKey: "b", PriceColumn: 2, BasePrice: 200, Discount: 20, Price: 160}}
	if err := ApplyBudgetPricing(sheet, 3, "skin_synergy", rows, entries); err == nil {
		t.Fatal("conflicting shared discounts accepted")
	}
	if sheet.Value(1, 2) != "0" {
		t.Fatal("shared discount mutated before conflict check")
	}
}

func TestBudgetCellPriceMatchesMainRestrictedEvaluation(t *testing.T) {
	cases := []struct {
		name, formula string
		expected      float64
	}{{"arithmetic", "ROUND(B2*(1-30%),2)", 70}, {"sum", "SUM(B2:B3)", 300}, {"sumproduct", "SUMPRODUCT(B2:B3,C2:C3)", 800}, {"unsupported cache", "IF(B2>0,B2,0)", 42}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			book, err := xlsx.NewCodec().(spreadsheet.TableCodec).NewTable("Blank", []string{"A", "B", "C", "D"}, [][]any{{"", 100, 2, 42}, {"", 200, 3, 0}})
			if err != nil {
				t.Fatal(err)
			}
			sheet, _ := book.Sheet("Blank")
			sheet.(spreadsheet.FormulaWriter).SetFormula(2, 4, tc.formula, 42)
			actual, ok := BudgetCellPrice(sheet, 2, 4)
			if !ok || actual != tc.expected {
				t.Fatalf("got %v (%v), want %v", actual, ok, tc.expected)
			}
		})
	}
}

func TestBudgetPriceRequiresPriceForPostEditQuantity(t *testing.T) {
	t.Parallel()
	book, err := xlsx.NewCodec().(spreadsheet.TableCodec).NewTable("Blank", []string{"Артикул", "Цена", "Количество"}, [][]any{{"A", 0, 3}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := book.Sheet("Blank")
	err = ApplyBudgetPricing(sheet, 1, "skin_synergy", []ReportRow{{Key: "a", BlankRow: 2, BlankQuantityColumn: 3, Inserted: new(0.0)}}, []BudgetPricing{{RowKey: "a", PriceColumn: 2, Price: 0}})
	if err == nil {
		t.Fatal("newly ordered row must have a price")
	}
}

func TestBudgetMetadataRefreshesAfterCompanyDiscount(t *testing.T) {
	t.Parallel()
	book, err := xlsx.NewCodec().(spreadsheet.TableCodec).NewTable("Blank", []string{"", ""}, [][]any{{"", ""}, {"Артикул", "Наименование", "Форма выпуска", "Количество", "Цена", "Цена со скидкой", "Сумма"}, {"A", "Крем", "мл", 2, 100, 100, 200}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, _ := book.Sheet("Blank")
	sheet.SetNumber(1, 2, .3)
	writer := sheet.(spreadsheet.FormulaWriter)
	writer.SetFormula(4, 6, "ROUND(E4*(1-$B$1),2)", 100)
	writer.SetFormula(4, 7, "F4*D4", 200)
	rows := []ReportRow{{Key: "a", BlankRow: 4, BlankQuantityColumn: 4, BudgetPrice: 100}}
	if err := RefreshBudgetData(book, brand.Rule("skin_synergy"), rows); err != nil {
		t.Fatal(err)
	}
	if rows[0].BudgetPrice != 70 || sheet.Value(4, 7) != "140" {
		t.Fatalf("stale profile price: %+v total=%s", rows, sheet.Value(4, 7))
	}
}
