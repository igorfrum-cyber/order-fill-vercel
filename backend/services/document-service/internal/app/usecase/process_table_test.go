package usecase

import (
	"context"
	"order-fill/backend/services/document-service/internal/app/port"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"testing"
)

func TestProcessTableCompletesWithoutSupplierBlank(t *testing.T) {
	t.Parallel()
	grid := [][]string{
		{"Период: 01.08.2025 - 31.07.2026", "", `Номенклатура В группе "Ангиофарм " И`},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"", "", "", "Сентябрь 2025", "Октябрь 2025", "Итого"},
		{"№", "Артикул", "Товар", "Количество", "Количество", "Количество", "Сумма выручки", "% выручки", "Кумулятивный %", "Категория", "Среднее в месяц", "Количество прошлый период", "Целевой запас", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"1", "A100", "Крем 50 мл", "10", "12", "22", "1000", "", "", "B", "11", "5", "30", "10", "4", "0", "7", "согласовано"},
	}
	store, jobs, reports := newStorageWithInputs(), &fakeJobStore{}, &fakeReportStore{}
	processor := newProcessor(store, jobs, reports)
	processor.codec = fakeCodec{grids: map[string][][]string{"source": grid}}
	message := processMessage()
	message.Type = "table_recalculate"
	message.Inputs = message.Inputs[:1]
	if err := processor.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if jobs.failCode != "" || jobs.statuses[len(jobs.statuses)-1] != "completed" || len(jobs.outputs) != 1 {
		t.Fatalf("job=%+v", jobs)
	}
	if reports.rows[0].OrderedFact == nil || *reports.rows[0].OrderedFact != 7 || reports.rows[0].SourceComment != "согласовано" {
		t.Fatalf("fact lost: %+v", reports.rows)
	}
	if jobs.outputs[0].Name != orderfill.SourceOutputFileName("Заказ.xlsx") {
		t.Fatalf("wrong output %+v", jobs.outputs)
	}
	_ = port.StageProcess
}
