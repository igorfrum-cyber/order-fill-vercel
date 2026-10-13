package north

import (
	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
	"strconv"
	"testing"
)

func TestNorthSothysReadsAndWritesSeparateBlocks(t *testing.T) {
	book := mutableNorthBook{rows: [][]string{{"", "English", "Русский", "Артикул", "Объем", "Ед.", "Цена", "Заказ", "Артикул", "Объем", "Ед.", "Цена", "Заказ"}, {"", "Cream", "Крем", "110-15", "50", "мл", "100", "2", "110-20", "100", "мл", "200", "3"}}}
	needs, err := NeedsFromBlank(&book, brand.Rule("sothys"), "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(needs) != 2 || needs[0].Article != "110-15" || needs[1].Article != "110-20" || needs[0].Qty != 2 || needs[1].Qty != 3 || needs[0].Price != 100 || needs[1].Price != 200 {
		t.Fatalf("needs = %+v", needs)
	}
	if err := WriteSupplierQuantities(&book, brand.Rule("sothys"), "", []SupplierLine{{Key: "110-15", Quantity: 8}, {Key: "110-20", Quantity: 9}, {Key: "missing", Quantity: 4}}); err != nil {
		t.Fatal(err)
	}
	if book.Value(2, 8) != "8" || book.Value(2, 13) != "9" || len(book.rows) != 2 {
		t.Fatal("main writes specialized positions only, without appending")
	}
}

func TestNorthNovacutanConvertsSupplierBoxesToNeedPieces(t *testing.T) {
	book := mutableNorthBook{rows: [][]string{{"NOVACUTAN"}, {"Описание", "Цена", "Заказ от", "Кол-во"}, {"EYE FILLER MASK NOVACUTAN", "100", "10", "3"}}}
	needs, err := NeedsFromBlank(&book, brand.Rule("novacutan"), "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(needs) != 1 || needs[0].Article != "novacutan:mask-eye" || needs[0].Qty != 15 {
		t.Fatalf("needs = %+v", needs)
	}
	if err := WriteSupplierQuantities(&book, brand.Rule("novacutan"), "", []SupplierLine{{Key: "novacutan:mask-eye", Quantity: 4}}); err != nil {
		t.Fatal(err)
	}
	if book.Value(3, 4) != "4" {
		t.Fatal("supplier quantity must remain boxes")
	}
}

func TestNorthNovacutanAvailabilityUsesProductNameKey(t *testing.T) {
	book := gridBook([][]string{{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"}, {"N1", "EYE FILLER MASK NOVACUTAN", "4", "10", "0", "", ""}})
	stock, err := StockFromSource(book, brand.Rule("novacutan"))
	if err != nil {
		t.Fatal(err)
	}
	if len(stock) != 1 || stock[0].Article != "novacutan:mask-eye" {
		t.Fatalf("stock = %+v", stock)
	}
}

type mutableNorthBook struct{ rows [][]string }

func (b *mutableNorthBook) Sheets() []spreadsheet.Sheet            { return []spreadsheet.Sheet{b} }
func (b *mutableNorthBook) Sheet(string) (spreadsheet.Sheet, bool) { return b, true }
func (b *mutableNorthBook) Save() ([]byte, error)                  { return nil, nil }
func (b *mutableNorthBook) Name() string                           { return "Лист" }
func (b *mutableNorthBook) Bounds() spreadsheet.Bounds             { return gridSheet(b.rows).Bounds() }
func (b *mutableNorthBook) Value(row, col int) string              { return gridSheet(b.rows).Value(row, col) }
func (b *mutableNorthBook) SetText(row, col int, value string) {
	for len(b.rows) < row {
		b.rows = append(b.rows, nil)
	}
	for len(b.rows[row-1]) < col {
		b.rows[row-1] = append(b.rows[row-1], "")
	}
	b.rows[row-1][col-1] = value
}
func (b *mutableNorthBook) SetNumber(row, col int, value float64) {
	b.SetText(row, col, strconv.FormatFloat(value, 'f', -1, 64))
}
func (b *mutableNorthBook) ClearValue(row, col int) { b.SetText(row, col, "") }
func (b *mutableNorthBook) DeleteRows([]int)        {}

func TestNorthSothysDiscountUpdatesEveryBlockPrice(t *testing.T) {
	book := mutableNorthBook{rows: [][]string{{"", "English", "Русский", "Артикул", "Объем", "Ед.", "Цена", "Заказ", "Артикул", "Объем", "Ед.", "Цена", "Заказ"}, {"", "Cream", "Крем", "110-15", "50", "мл", "100", "2", "110-20", "100", "мл", "200", "3"}}}
	if err := ApplySupplierDiscount(&book, brand.Rule("sothys"), 10); err != nil {
		t.Fatal(err)
	}
	if book.Value(2, 7) != "90" || book.Value(2, 12) != "180" {
		t.Fatalf("prices = %q / %q", book.Value(2, 7), book.Value(2, 12))
	}
}
func (b *mutableNorthBook) Formulas() []spreadsheet.Formula { return nil }
func (b *mutableNorthBook) SetFormula(row, col int, _ string, value float64) {
	b.SetNumber(row, col, value)
}
func (b *mutableNorthBook) SetFormulaCache(row, col int, value float64) { b.SetNumber(row, col, value) }

func TestNorthAvailabilitySkipsTotalAndEmptyRows(t *testing.T) {
	book := gridBook([][]string{{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"}, {"N1", "NOVACUTAN SBIO", "4", "10", "0", "", ""}, {"Итого", "", "4", "10"}, {"", "", "", ""}})
	stock, err := StockFromSource(book, brand.Rule("novacutan"))
	if err != nil {
		t.Fatal(err)
	}
	if len(stock) != 1 {
		t.Fatalf("stock rows = %+v", stock)
	}
}

func TestNorthNovacutanPreservesSupplierMinimumOverride(t *testing.T) {
	book := mutableNorthBook{rows: [][]string{{"NOVACUTAN"}, {"Описание", "Цена", "Заказ от", "Кол-во"}, {"EYE FILLER MASK NOVACUTAN", "100", "25", "3"}}}
	needs, err := NeedsFromBlank(&book, brand.Rule("novacutan"), "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(needs) != 1 || needs[0].Minimum != 25 || needs[0].SupplierUnitSize != 5 {
		t.Fatalf("metadata = %+v", needs)
	}
}

func TestNorthGenericPreservesSupplierBoxMetadata(t *testing.T) {
	book := gridBook([][]string{{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке", "Цена"}, {"A1", "Крем", "50 мл", "3", "12", "100"}})
	needs, err := NeedsFromBlank(book, brand.Rule("angiopharm"), "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(needs) != 1 || !needs[0].HasBoxSize || needs[0].BoxSize != 12 || needs[0].SupplierUnitSize != 1 {
		t.Fatalf("metadata = %+v", needs)
	}
}
