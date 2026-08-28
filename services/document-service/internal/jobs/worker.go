package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

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
	Logger         *slog.Logger
	Metrics        Metrics
}

type Worker struct {
	storage        Storage
	reportWriter   ReportWriter
	orderProcessor orderfill.Processor
	logger         *slog.Logger
	metrics        Metrics
}

type Metrics interface {
	AddJobCompleted(durationMS int64)
	AddJobFailed(durationMS int64)
}

func NewWorker(config WorkerConfig) *Worker {
	return &Worker{
		storage:        config.Storage,
		reportWriter:   config.ReportWriter,
		orderProcessor: config.OrderProcessor,
		logger:         defaultLogger(config.Logger),
		metrics:        config.Metrics,
	}
}

func (w *Worker) Handle(ctx context.Context, message JobMessage) error {
	startedAt := time.Now()
	if message.JobID == "" {
		err := fmt.Errorf("job id is required")
		w.observeFailure(ctx, message.JobID, "job_invalid", startedAt, "invalid_job", err)
		return err
	}
	files, err := w.loadInputs(ctx, message.InputKeys)
	if err != nil {
		w.observeFailure(ctx, message.JobID, "load_inputs_failed", startedAt, "storage_error", err)
		return err
	}
	switch message.Type {
	case JobTypeOrderFill:
		if err := w.processOrderFill(ctx, message.JobID, files); err != nil {
			w.observeFailure(ctx, message.JobID, "process_job_failed", startedAt, "processing_error", err)
			return err
		}
		durationMS := time.Since(startedAt).Milliseconds()
		if w.metrics != nil {
			w.metrics.AddJobCompleted(durationMS)
		}
		w.logger.InfoContext(ctx, "document job completed",
			"service", "document-service",
			"job_id", message.JobID,
			"event", "job_completed",
			"duration_ms", durationMS,
			"error_code", "",
			"type", message.Type,
		)
		return nil
	default:
		err := fmt.Errorf("unsupported job type %q", message.Type)
		w.observeFailure(ctx, message.JobID, "unsupported_job_type", startedAt, "unsupported_job_type", err)
		return err
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

func defaultLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}

func (w *Worker) observeFailure(ctx context.Context, jobID string, event string, startedAt time.Time, errorCode string, err error) {
	durationMS := time.Since(startedAt).Milliseconds()
	if w.metrics != nil {
		w.metrics.AddJobFailed(durationMS)
	}
	w.logger.ErrorContext(ctx, "document job failed",
		"service", "document-service",
		"job_id", jobID,
		"event", event,
		"duration_ms", durationMS,
		"error_code", errorCode,
		"error", err,
	)
}
