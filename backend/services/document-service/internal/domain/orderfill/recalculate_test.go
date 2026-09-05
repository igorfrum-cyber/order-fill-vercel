package orderfill

import "testing"

// chzSourceGrid is a 1C export with the full ABC analysis block, which is what
// switches the engine into recalculation mode. Article AA04 is listed twice:
// once normally and once as its "ЧЗ" clone, the pair the merge folds together.
func chzSourceGrid() [][]string {
	return [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"", "", "", "Сентябрь 2025", "Октябрь 2025", "Итого"},
		{
			"№", "Артикул", "Товар", "Количество", "Количество", "Количество",
			"Сумма выручки", "% выручки", "Кумулятивный %", "Категория",
			"Среднее в месяц", "Количество прошлый период", "Целевой запас",
			"Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий",
		},
		{"1", "AA04", "АН Сыворотка 30 мл", "10", "12", "22", "1000", "", "", "", "", "5", "", "0", "4", "0", "", ""},
		{"2", "AA04", "ЧЗ АН Сыворотка 30 мл", "3", "5", "8", "400", "", "", "", "", "2", "", "0", "2", "0", "", ""},
		{"3", "BB01", "АН Крем 50 мл", "4", "4", "8", "200", "", "", "", "", "3", "", "0", "1", "0", "", ""},
	}
}

func chzBlankGrid() [][]string {
	return [][]string{
		{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
		{"AA04", "Сыворотка", "30 мл", "", "3"},
		{"BB01", "Крем", "50 мл", "", "3"},
	}
}

func chzFixture(t *testing.T) Result {
	t.Helper()
	result, err := Fill(FillCommand{
		Source:     newFakeWorkbook("Заказ", chzSourceGrid()),
		Blank:      newFakeWorkbook("Бланк", chzBlankGrid()),
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

func TestChzMergeRemovesTheCloneRow(t *testing.T) {
	result := chzFixture(t)
	sheet, _ := result.Source.Sheet("Заказ")

	if got := sheet.Value(5, 3); got != "ЧЗ + АН Сыворотка 30 мл" {
		t.Fatalf("merged row name = %q, want the combined ЧЗ name", got)
	}
	if got := sheet.Value(6, 2); got != "BB01" {
		t.Fatalf("row 6 article = %q, want BB01 moved up into the deleted clone row", got)
	}
	if got := sheet.Bounds().MaxRow; got != 6 {
		t.Fatalf("sheet has %d rows, want 6 after the clone row was deleted", got)
	}
}

func TestChzMergeSumsTheCloneQuantities(t *testing.T) {
	result := chzFixture(t)
	sheet, _ := result.Source.Sheet("Заказ")

	cases := []struct {
		name   string
		column int
		want   string
	}{
		{name: "september sales", column: 4, want: "13"},
		{name: "october sales", column: 5, want: "17"},
		{name: "stock", column: 15, want: "6"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := sheet.Value(5, testCase.column); got != testCase.want {
				t.Fatalf("merged %s = %q, want %q", testCase.name, got, testCase.want)
			}
		})
	}
}

// A clone row left behind by the merge makes its article look like two
// competing source positions, which is how a clean export ended up reported as
// hundreds of duplicates.
func TestChzMergeLeavesNoDuplicateArticles(t *testing.T) {
	result := chzFixture(t)

	if result.Summary.Duplicates != 0 {
		t.Fatalf("summary reports %d duplicates, want none", result.Summary.Duplicates)
	}
	for _, row := range result.Rows {
		if row.Status == StatusSourceDuplicate {
			t.Fatalf("report row %q is a source duplicate, want none", row.Key)
		}
		if row.Duplicate {
			t.Fatalf("report row %q is flagged as a duplicate, want none", row.Key)
		}
	}
}
