package jobs

import (
	"context"
	"fmt"
	"path/filepath"

	"order-fill/services/document-service/internal/orderfill"
)

type JobType string

const (
	JobTypeOrderFill  JobType = "order_fill"
	JobTypeNorthMerge JobType = "north_merge"
)

type JobMessage struct {
	JobID     string
	Type      JobType
	InputKeys []string
}

type Storage interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

type ReportWriter interface {
	Save(ctx context.Context, jobID string, report orderfill.Report) error
}

type WorkerConfig struct {
	Storage        Storage
	ReportWriter   ReportWriter
	OrderProcessor orderfill.Processor
}

type Worker struct {
	storage        Storage
	reportWriter   ReportWriter
	orderProcessor orderfill.Processor
}

func NewWorker(config WorkerConfig) *Worker {
	return &Worker{
		storage:        config.Storage,
		reportWriter:   config.ReportWriter,
		orderProcessor: config.OrderProcessor,
	}
}

func (w *Worker) Handle(ctx context.Context, message JobMessage) error {
	if message.JobID == "" {
		return fmt.Errorf("job id is required")
	}
	files, err := w.loadInputs(ctx, message.InputKeys)
	if err != nil {
		return err
	}
	switch message.Type {
	case JobTypeOrderFill:
		return w.processOrderFill(ctx, message.JobID, files)
	default:
		return fmt.Errorf("unsupported job type %q", message.Type)
	}
}

func (w *Worker) loadInputs(ctx context.Context, keys []string) ([]orderfill.InputFile, error) {
	if w.storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	files := make([]orderfill.InputFile, 0, len(keys))
	for _, key := range keys {
		content, err := w.storage.Get(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("load input %s: %w", key, err)
		}
		files = append(files, orderfill.InputFile{Name: filepath.Base(key), Content: content})
	}
	return files, nil
}

func (w *Worker) processOrderFill(ctx context.Context, jobID string, files []orderfill.InputFile) error {
	if w.orderProcessor == nil {
		return fmt.Errorf("order processor is required")
	}
	if w.reportWriter == nil {
		return fmt.Errorf("report writer is required")
	}
	report, err := w.orderProcessor.Process(ctx, files)
	if err != nil {
		return fmt.Errorf("process order-fill job: %w", err)
	}
	if err := w.reportWriter.Save(ctx, jobID, report); err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	return nil
}
