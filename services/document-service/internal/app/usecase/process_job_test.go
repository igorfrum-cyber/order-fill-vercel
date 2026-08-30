package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"order-fill/services/document-service/internal/app/port"
	"order-fill/services/document-service/internal/domain/orderfill"
	"order-fill/services/document-service/internal/domain/spreadsheet"
)

// The fakes below stand in for MinIO, PostgreSQL and the xlsx adapter so the
// orchestration can be tested without any infrastructure.

type fakeSheet struct {
	name  string
	cells map[[2]int]string
}

func (s *fakeSheet) Name() string { return s.name }

func (s *fakeSheet) Bounds() spreadsheet.Bounds {
	bounds := spreadsheet.Bounds{}
	for reference := range s.cells {
		bounds.MaxRow = max(bounds.MaxRow, reference[0])
		bounds.MaxColumn = max(bounds.MaxColumn, reference[1])
	}
	return bounds
}

func (s *fakeSheet) Value(row int, column int) string { return s.cells[[2]int{row, column}] }

func (s *fakeSheet) SetNumber(row int, column int, value float64) {
	s.cells[[2]int{row, column}] = strconv.FormatFloat(value, 'f', -1, 64)
}

func (s *fakeSheet) ClearValue(row int, column int) { delete(s.cells, [2]int{row, column}) }

func (s *fakeSheet) SetText(row int, column int, value string) {
	if value == "" {
		delete(s.cells, [2]int{row, column})
		return
	}
	s.cells[[2]int{row, column}] = value
}

func (s *fakeSheet) DeleteRows(rows []int) {
	removed := map[int]bool{}
	for _, row := range rows {
		removed[row] = true
	}
	moved := map[[2]int]string{}
	for reference, value := range s.cells {
		if removed[reference[0]] {
			continue
		}
		above := 0
		for _, row := range rows {
			if row < reference[0] {
				above++
			}
		}
		moved[[2]int{reference[0] - above, reference[1]}] = value
	}
	s.cells = moved
}

type fakeWorkbook struct {
	sheet *fakeSheet
	saved []byte
}

func (w *fakeWorkbook) Sheets() []spreadsheet.Sheet { return []spreadsheet.Sheet{w.sheet} }

func (w *fakeWorkbook) Sheet(name string) (spreadsheet.Sheet, bool) {
	if w.sheet.name == name {
		return w.sheet, true
	}
	return nil, false
}

func (w *fakeWorkbook) Save() ([]byte, error) { return w.saved, nil }

type fakeCodec struct {
	grids map[string][][]string
}

func (c fakeCodec) Load(content []byte) (spreadsheet.Workbook, error) {
	grid, ok := c.grids[string(content)]
	if !ok {
		return nil, errors.New("unknown workbook")
	}
	sheet := &fakeSheet{name: "Лист1", cells: map[[2]int]string{}}
	for rowIndex, row := range grid {
		for columnIndex, value := range row {
			if value != "" {
				sheet.cells[[2]int{rowIndex + 1, columnIndex + 1}] = value
			}
		}
	}
	return &fakeWorkbook{sheet: sheet, saved: content}, nil
}

type fakeStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (s *fakeStorage) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.objects[key]
	if !ok {
		return nil, errors.New("missing object " + key)
	}
	return content, nil
}

func (s *fakeStorage) Put(_ context.Context, key string, _ string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = content
	return nil
}

type fakeJobStore struct {
	mu       sync.Mutex
	statuses []string
	outputs  []port.OutputFile
	failCode string
	failText string
	progress []progressNote
}

type progressNote struct {
	fraction float64
	message  string
}

func (s *fakeJobStore) MarkProcessing(context.Context, string, time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses = append(s.statuses, "processing")
	return nil
}

func (s *fakeJobStore) MarkFailed(_ context.Context, _ string, code string, message string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses = append(s.statuses, "failed")
	s.failCode = code
	s.failText = message
	return nil
}

func (s *fakeJobStore) SaveResult(_ context.Context, _ string, status string, outputs []port.OutputFile, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses = append(s.statuses, status)
	s.outputs = outputs
	return nil
}

func (s *fakeJobStore) SetProgress(_ context.Context, _ string, fraction float64, message string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress = append(s.progress, progressNote{fraction: fraction, message: message})
	return nil
}

func (s *fakeJobStore) Outputs(context.Context, string) ([]port.OutputFile, error) {
	return s.outputs, nil
}

type fakeReportStore struct {
	summary orderfill.Summary
	rows    []orderfill.ReportRow
	saved   int
}

func (s *fakeReportStore) Save(_ context.Context, _ string, summary orderfill.Summary, rows []orderfill.ReportRow, _ time.Time) error {
	s.summary = summary
	s.rows = rows
	s.saved++
	return nil
}

func (s *fakeReportStore) Load(context.Context, string) (orderfill.Summary, []orderfill.ReportRow, error) {
	return s.summary, s.rows, nil
}

func testGrids() map[string][][]string {
	return map[string][][]string{
		"source": {
			{"Период: 01.08.2025 - 31.07.2026"},
			{"Прошлый период: 01.08.2025 - 31.10.2025"},
			{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
			{"A100", "Крем для лица 50 мл", "10", "0", "0", "", ""},
		},
		"blank": {
			{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
			{"A100", "Крем для лица", "50 мл", "", "3"},
		},
	}
}

func newProcessor(storage *fakeStorage, jobs *fakeJobStore, reports *fakeReportStore) *ProcessJob {
	now := func() time.Time { return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC) }
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewProcessJob(fakeCodec{grids: testGrids()}, storage, jobs, reports, now, logger, nil)
}

func processMessage() port.JobMessage {
	return port.JobMessage{
		JobID:      "job-1",
		Type:       "order_fill",
		Stage:      port.StageProcess,
		Brand:      "angiopharm",
		OrderMonth: "2026-09",
		Inputs: []port.MessageFile{
			{Role: port.RoleSource, Name: "Заказ.xlsx", StorageKey: "jobs/job-1/inputs/0-source.xlsx"},
			{Role: port.RoleBlank, Name: "Бланк.xlsx", StorageKey: "jobs/job-1/inputs/1-blank.xlsx"},
		},
	}
}

func newStorageWithInputs() *fakeStorage {
	return &fakeStorage{objects: map[string][]byte{
		"jobs/job-1/inputs/0-source.xlsx": []byte("source"),
		"jobs/job-1/inputs/1-blank.xlsx":  []byte("blank"),
	}}
}

func TestProcessJobFillsTheBlankAndPublishesTheReport(t *testing.T) {
	storage := newStorageWithInputs()
	jobs := &fakeJobStore{}
	reports := &fakeReportStore{}

	if err := newProcessor(storage, jobs, reports).Handle(context.Background(), processMessage()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := jobs.statuses; len(got) != 2 || got[0] != "processing" || got[1] != "needs_review" {
		t.Fatalf("unexpected status transitions %v", got)
	}
	if len(jobs.outputs) != 2 {
		t.Fatalf("expected a filled blank and a filled source, got %d outputs", len(jobs.outputs))
	}
	if jobs.outputs[0].ID != "output-1" || jobs.outputs[1].ID != "output-2" {
		t.Fatalf("outputs need URL safe identifiers, got %q and %q", jobs.outputs[0].ID, jobs.outputs[1].ID)
	}
	if jobs.outputs[0].Name != "Бланк заполненный.xlsx" {
		t.Fatalf("unexpected output name %q", jobs.outputs[0].Name)
	}
	if _, stored := storage.objects[jobs.outputs[0].StorageKey]; !stored {
		t.Fatal("the generated blank must be written to object storage")
	}
	if reports.saved != 1 || len(reports.rows) == 0 {
		t.Fatalf("expected the report to be stored once with rows, got %d saves", reports.saved)
	}
	if reports.summary.Filled != 1 {
		t.Fatalf("expected one filled position, got %d", reports.summary.Filled)
	}
	if len(jobs.progress) == 0 {
		t.Fatal("the worker must publish progress while the job is running")
	}
	var last float64
	for _, note := range jobs.progress {
		if note.fraction+1e-9 < last {
			t.Fatalf("progress went backwards: %v -> %v", last, note.fraction)
		}
		last = note.fraction
		if note.message == "" {
			t.Fatal("every progress update needs a user-facing message")
		}
	}
}

func TestProcessJobRecordsUserFacingFailureWithoutRetrying(t *testing.T) {
	storage := newStorageWithInputs()
	jobs := &fakeJobStore{}
	message := processMessage()
	message.OrderMonth = "2026-12"

	err := newProcessor(storage, jobs, &fakeReportStore{}).Handle(context.Background(), message)
	if err != nil {
		t.Fatalf("a user error must not be returned for retry, got %v", err)
	}
	if jobs.failCode != "invalid_input" {
		t.Fatalf("expected an invalid_input failure, got %q", jobs.failCode)
	}
	if jobs.failText == "" {
		t.Fatal("the user needs an explanation of what to fix")
	}
}

func TestProcessJobFinalizeAppliesEditsAndCompletesTheJob(t *testing.T) {
	storage := newStorageWithInputs()
	jobs := &fakeJobStore{}
	reports := &fakeReportStore{}
	processor := newProcessor(storage, jobs, reports)

	if err := processor.Handle(context.Background(), processMessage()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	message := processMessage()
	message.Stage = port.StageFinalize
	message.Edits = []port.MessageEdit{{Key: "blank-1:2", Value: "7", Comment: "договорились с поставщиком"}}

	if err := processor.Handle(context.Background(), message); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	last := jobs.statuses[len(jobs.statuses)-1]
	if last != "completed" {
		t.Fatalf("expected the job to complete, got %q", last)
	}
}

func TestProcessJobRejectsAMessageWithoutABlank(t *testing.T) {
	jobs := &fakeJobStore{}
	message := processMessage()
	message.Inputs = message.Inputs[:1]

	if err := newProcessor(newStorageWithInputs(), jobs, &fakeReportStore{}).Handle(context.Background(), message); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jobs.failCode != "invalid_input" {
		t.Fatalf("expected an invalid_input failure, got %q", jobs.failCode)
	}
}
