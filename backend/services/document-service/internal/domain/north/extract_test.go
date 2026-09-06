package north

import (
	"testing"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

func TestCityFromFileName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, key string
		ok        bool
	}{
		{"Сургут.xlsx", "surgut", true},
		{"blank-urengoy-HOME.xlsx", "urengoy", true},
		{"Тюмень остатки.xlsx", "tyumen", true},
		{"Нижневартовск.xlsx", "nizhnevartovsk", true},
		{"blank.xlsx", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key, _, ok := CityFromFileName(tc.name)
			if ok != tc.ok || key != tc.key {
				t.Fatalf("key=%s ok=%v want %s/%v", key, ok, tc.key, tc.ok)
			}
		})
	}
}

func TestStockFromSourceReadsTarget(t *testing.T) {
	t.Parallel()
	got, err := StockFromSource(gridBook([][]string{
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий", "Целевой запас"},
		{"A1", "Cream", "0", "20", "2", "", "", "5"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Stock != 20 || got[0].InTransit != 2 || got[0].Target != 5 {
		t.Fatalf("stock=%+v", got)
	}
}

func TestNeedsFromBlankUsesRule(t *testing.T) {
	t.Parallel()
	got, err := NeedsFromBlank(gridBook([][]string{
		{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
		{"A1", "Cream", "50 мл", "10", "3"},
	}), brand.Rule("angiopharm"), "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].City != "surgut" || got[0].Qty != 10 {
		t.Fatalf("needs=%+v", got)
	}
}

type gridBook [][]string

func (g gridBook) Sheets() []spreadsheet.Sheet { return []spreadsheet.Sheet{gridSheet(g)} }

func (g gridBook) Sheet(string) (spreadsheet.Sheet, bool) { return gridSheet(g), true }

func (g gridBook) Save() ([]byte, error) { return nil, nil }

type gridSheet [][]string

func (s gridSheet) Name() string { return "Лист1" }

func (s gridSheet) Bounds() spreadsheet.Bounds {
	b := spreadsheet.Bounds{}
	for i, row := range s {
		b.MaxRow = i + 1
		if c := len(row); c > b.MaxColumn {
			b.MaxColumn = c
		}
	}
	return b
}

func (s gridSheet) Value(row, column int) string {
	if row < 1 || row > len(s) || column < 1 || column > len(s[row-1]) {
		return ""
	}
	return s[row-1][column-1]
}

func (gridSheet) SetNumber(int, int, float64) {}
func (gridSheet) ClearValue(int, int)         {}
func (gridSheet) SetText(int, int, string)    {}
func (gridSheet) DeleteRows([]int)            {}
