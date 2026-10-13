package usecase

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"order-fill/backend/services/document-service/internal/app/port"
	"order-fill/backend/services/document-service/internal/domain/north"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

func TestProcessNorthRecalculatesCombinedMonthlyHistory(t *testing.T) {
	grids := testGrids()
	grids["north-blank"] = [][]string{{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"}, {"A1", "Cream", "50 мл", "10", "3"}}
	// Main's novelty rule: office [10,0] target12.5 + warehouse [0,10]
	// target17.5 differs from combined [10,10] target17.5.
	grids["office"] = northFullABCGrid("10", "0", "12.5", "20", "4")
	grids["warehouse"] = northFullABCGrid("0", "10", "17.5", "8", "7")
	storage := &fakeStorage{objects: map[string][]byte{"blank": []byte("north-blank"), "office": []byte("office"), "warehouse": []byte("warehouse")}}
	jobs := &fakeJobStore{}
	calc := &pairedNorthCalc{}
	processor := NewProcessJob(fakeCodec{grids: grids}, storage, jobs, &fakeReportStore{}, func() time.Time { return time.Now() }, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, calc, exactJobMatcher{}, identityBrand{})
	if err := processor.Handle(t.Context(), port.JobMessage{JobID: "job-pair", Type: "north_merge", Brand: "angiopharm", Inputs: []port.MessageFile{{Role: port.RoleBlank, Name: "Сургут.xlsx", StorageKey: "blank"}, {Role: port.RoleSource, Name: "Тюмень.xlsx", StorageKey: "office"}, {Role: port.RoleWarehouse, Name: "Склад.xlsx", StorageKey: "warehouse"}}}); err != nil {
		t.Fatal(err)
	}
	if jobs.failCode != "" {
		t.Fatalf("worker failed: %s", jobs.failCode)
	}
	if calc.recommendCalls != 1 || len(calc.sales) != 2 || calc.sales[0] != 10 || calc.sales[1] != 10 {
		t.Fatalf("recommendation must consume combined history once: calls %d sales %v", calc.recommendCalls, calc.sales)
	}
	if len(calc.stock) != 1 || calc.stock[0].Target != 17.5 || calc.stock[0].Stock != 28 || calc.stock[0].WarehouseStock != 8 || !calc.stock[0].HasWarehouseStock {
		t.Fatalf("combined availability = %+v", calc.stock)
	}
	for _, need := range calc.needs {
		if need.City == "tyumen" && need.Qty != 0 {
			t.Fatalf("main clears paired actual orders before recomputation; got %+v", need)
		}
	}
}

func northFullABCGrid(first, second, target, stock, fact string) [][]string {
	return [][]string{{"Период: 01.08.2025 - 31.07.2026"}, {"Прошлый период: 01.08.2025 - 31.10.2025"}, {"", "", "Август 2025", "Сентябрь 2025", "Итого"}, {"Артикул", "Товар", "Количество", "Количество", "Количество", "Сумма выручки", "% выручки", "Кумулятивный %", "Категория", "Среднее в месяц", "Количество прошлый период", "Целевой запас", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"}, {"A1", "Cream", first, second, "10", "100", "", "", "C", "5", "1", target, "1", stock, "0", fact, ""}}
}

type pairedNorthCalc struct {
	fakeNorthCalc
	recommendCalls int
	sales          []float64
}

func (f *pairedNorthCalc) Recommend(_ context.Context, _ string, _ string, _ float64, rows []orderfill.RecommendRow) ([]orderfill.RecommendRow, error) {
	f.recommendCalls++
	for i := range rows {
		f.sales = append([]float64{}, rows[i].MonthlySales...)
		rows[i].TargetStock = 17.5
		rows[i].Recommended = 0
		rows[i].MonthlyNeed = 15
		rows[i].TotalQuantity = 20
	}
	return rows, nil
}

func TestCombineTyumenLocationsKeepsEmbeddedWarehouseCap(t *testing.T) {
	_, rows := combineTyumenLocations([]north.Stock{{Article: "A1", Stock: 100, Target: 10, WarehouseStock: 8, WarehouseTransit: 2, HasWarehouseStock: true}}, nil)
	if len(rows) != 1 || !rows[0].HasWarehouseStock || rows[0].WarehouseStock != 8 || rows[0].WarehouseTransit != 2 {
		t.Fatalf("embedded cap = %+v", rows)
	}
}
