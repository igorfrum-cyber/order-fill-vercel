package jobs

import (
	"context"
	"testing"

	"order-fill/services/document-service/internal/orderfill"
	"order-fill/services/document-service/internal/reports"
)

func TestWorkerProcessesOrderFillJobAndStoresReport(t *testing.T) {
	storage := &recordingStorage{
		files: map[string][]byte{
			"source.xlsx": []byte("source"),
			"blank.xlsx":  []byte("blank"),
		},
	}
	repository := reports.NewMemoryRepository()
	processor := orderfill.ProcessorFunc(func(_ context.Context, files []orderfill.InputFile) (orderfill.Report, error) {
		if len(files) != 2 {
			t.Fatalf("expected two input files, got %d", len(files))
		}
		return orderfill.Report{Rows: []orderfill.ReportRow{{Key: "row-1", Status: "matched", Editable: false}}}, nil
	})
	worker := NewWorker(WorkerConfig{
		Storage:        storage,
		ReportWriter:   repository,
		OrderProcessor: processor,
	})

	err := worker.Handle(context.Background(), JobMessage{
		JobID:     "job-1",
		Type:      JobTypeOrderFill,
		InputKeys: []string{"source.xlsx", "blank.xlsx"},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(repository.Reports["job-1"].Rows) != 1 {
		t.Fatalf("expected stored report, got %#v", repository.Reports)
	}
}

type recordingStorage struct {
	files map[string][]byte
}

func (s *recordingStorage) Get(_ context.Context, key string) ([]byte, error) {
	return s.files[key], nil
}
