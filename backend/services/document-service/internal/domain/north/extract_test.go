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

func TestCityFromWorkbookPrefersWorkbookContent(t *testing.T) {
	t.Parallel()
	key, _, ok := CityFromWorkbook(gridBook([][]string{{"Заказ для склада Сургут"}}), "Тюмень.xlsx")
	if !ok || key != "surgut" {
		t.Fatalf("key=%q ok=%v", key, ok)
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

func TestStockFromSourceKeepsTyumenPlannedOrder(t *testing.T) {
	t.Parallel()
	got, err := StockFromSource(gridBook([][]string{
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"A1", "Cream", "7", "20", "2", "9", ""},
		{"A2", "Serum", "4", "10", "0", "", ""},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].HasActual || got[0].Actual != 9 || got[0].Recommended != 7 || got[1].HasActual || got[1].Recommended != 4 {
		t.Fatalf("stock=%+v", got)
	}
	if got[0].Target != 29 || got[1].Target != 14 {
		t.Fatalf("fallback targets=%+v", got)
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

func TestNeedsFromBlankKeepsVariantAndSkipsRepeatedHeader(t *testing.T) {
	t.Parallel()
	got, err := NeedsFromBlank(gridBook([][]string{
		{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
		{"A-1", "Cream", "50 мл", "10", "3"},
		{"", "Промо-продукция", "", "КОЛ-ВО", ""},
	}), brand.Rule("christina"), "surgut", "home")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Article != "home:A1" || got[0].Variant != "home" {
		t.Fatalf("needs=%+v", got)
	}
}

func TestNeedsFromBlankAttachesChristinaProffLine(t *testing.T) {
	t.Parallel()
	got, err := NeedsFromBlank(gridBook([][]string{
		{"Артикул", "Наименование", "Цена", "Кол-во"},
		{"", "MUSE", "", ""},
		{"CHR001", "Товар A", "100", "3"},
		{"CHR002", "Товар B", "100", "3"},
	}), brand.Rule("christina"), "surgut", "proff")
	if err != nil {
		t.Fatal(err)
	}
	var member *Need
	for i := range got {
		if got[i].ArticleRaw == "CHR001" {
			member = &got[i]
		}
	}
	if member == nil || member.Line == nil {
		t.Fatalf("CHR001 need must carry a christina line: %+v", got)
	}
	if member.Line.ID != "MUSE" || member.Line.Article != "CHR001" {
		t.Fatalf("line = %+v, want id MUSE / article CHR001", member.Line)
	}
	if len(member.Line.Required) != 2 || member.Line.Required[0] != "CHR001" || member.Line.Required[1] != "CHR002" {
		t.Fatalf("required = %v, want [CHR001 CHR002]", member.Line.Required)
	}
}

func TestNeedsFromBlankNoChristinaLineForOtherBrands(t *testing.T) {
	t.Parallel()
	got, err := NeedsFromBlank(gridBook([][]string{
		{"Артикул", "Наименование", "Кол-во"},
		{"CHR001", "Товар A", "3"},
	}), brand.Rule("angiopharm"), "surgut")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Line != nil {
		t.Fatalf("non-christina brand must not attach a line: %+v", got)
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
