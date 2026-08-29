// Package usecase contains the worker application logic: it orchestrates the
// domain engine, object storage and the job store without knowing about Redis,
// PostgreSQL, S3 or xlsx internals.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"order-fill/services/document-service/internal/app/port"
	"order-fill/services/document-service/internal/domain/orderfill"
	"order-fill/services/document-service/internal/domain/spreadsheet"
)

const workbookContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// ProcessJob runs one queue message end to end.
type ProcessJob struct {
	codec   spreadsheet.Codec
	storage port.ObjectStore
	jobs    port.JobStore
	reports port.ReportStore
	now     port.Clock
	logger  *slog.Logger
	metrics port.Metrics
}

func NewProcessJob(
	codec spreadsheet.Codec,
	storage port.ObjectStore,
	jobs port.JobStore,
	reports port.ReportStore,
	now port.Clock,
	logger *slog.Logger,
	metrics port.Metrics,
) *ProcessJob {
	return &ProcessJob{codec: codec, storage: storage, jobs: jobs, reports: reports, now: now, logger: logger, metrics: metrics}
}

// Handle processes a message and records the outcome on the job. A failure that
// the user caused is stored on the job and swallowed, because retrying it would
// not help; infrastructure failures are returned so the caller can retry.
func (u *ProcessJob) Handle(ctx context.Context, message port.JobMessage) error {
	startedAt := u.now()
	if message.JobID == "" {
		return fmt.Errorf("job id is required")
	}

	var err error
	switch message.Stage {
	case port.StageFinalize:
		err = u.finalize(ctx, message)
	default:
		err = u.process(ctx, message)
	}
	if err == nil {
		duration := millisSince(startedAt, u.now())
		if u.metrics != nil {
			u.metrics.AddJobCompleted(duration)
		}
		u.logger.InfoContext(ctx, "document job completed",
			"service", "document-service",
			"job_id", message.JobID,
			"event", "job_completed",
			"duration_ms", duration,
			"error_code", "",
			"stage", message.Stage,
		)
		return nil
	}

	code := "processing_error"
	if errors.Is(err, orderfill.ErrInvalidInput) {
		code = "invalid_input"
	}
	duration := millisSince(startedAt, u.now())
	if u.metrics != nil {
		u.metrics.AddJobFailed(duration)
	}
	u.logger.ErrorContext(ctx, "document job failed",
		"service", "document-service",
		"job_id", message.JobID,
		"event", "job_failed",
		"duration_ms", duration,
		"error_code", code,
		"error", err,
	)
	if markErr := u.jobs.MarkFailed(ctx, message.JobID, code, userMessage(err), u.now()); markErr != nil {
		return fmt.Errorf("mark job failed: %w (original error: %v)", markErr, err)
	}
	if code == "invalid_input" {
		return nil
	}
	return err
}

func (u *ProcessJob) process(ctx context.Context, message port.JobMessage) error {
	if err := u.jobs.MarkProcessing(ctx, message.JobID, u.now()); err != nil {
		return fmt.Errorf("mark job processing: %w", err)
	}
	if message.Type != "order_fill" {
		return fmt.Errorf("%w: тип задачи %q пока не поддерживается сервисом", orderfill.ErrInvalidInput, message.Type)
	}

	sourceInput, blankInputs, err := splitInputs(message.Inputs)
	if err != nil {
		return err
	}

	sourceWorkbook, err := u.loadWorkbook(ctx, sourceInput.StorageKey)
	if err != nil {
		return err
	}

	outputs := make([]port.OutputFile, 0, len(blankInputs)+1)
	rows := make([]orderfill.ReportRow, 0)
	summary := orderfill.Summary{}

	for index, blankInput := range blankInputs {
		blankWorkbook, err := u.loadWorkbook(ctx, blankInput.StorageKey)
		if err != nil {
			return err
		}
		result, err := orderfill.Fill(orderfill.FillCommand{
			Source:     sourceWorkbook,
			Blank:      blankWorkbook,
			OrderMonth: message.OrderMonth,
			Brand:      message.Brand,
			BlankID:    blankID(index),
			BlankLabel: blankInput.Name,
		})
		if err != nil {
			return err
		}
		rows = append(rows, result.Rows...)
		summary = mergeSummary(summary, result.Summary)

		output, err := u.saveWorkbook(ctx, message.JobID, blankWorkbook, orderfill.BlankOutputFileName(blankInput.Name, ""), "Скачать заполненный бланк")
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
	}

	sourceOutput, err := u.saveWorkbook(ctx, message.JobID, sourceWorkbook, orderfill.SourceOutputFileName(sourceInput.Name), "Скачать заполненную таблицу заказа")
	if err != nil {
		return err
	}
	outputs = append(outputs, sourceOutput)

	if err := u.reports.Save(ctx, message.JobID, summary, rows, u.now()); err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	if err := u.jobs.SaveResult(ctx, message.JobID, "needs_review", assignOutputIDs(outputs), u.now()); err != nil {
		return fmt.Errorf("save job result: %w", err)
	}
	return nil
}

func (u *ProcessJob) finalize(ctx context.Context, message port.JobMessage) error {
	summary, rows, err := u.reports.Load(ctx, message.JobID)
	if err != nil {
		return fmt.Errorf("load report: %w", err)
	}
	previousOutputs, err := u.jobs.Outputs(ctx, message.JobID)
	if err != nil {
		return fmt.Errorf("load job outputs: %w", err)
	}

	sourceInput, blankInputs, err := splitInputs(message.Inputs)
	if err != nil {
		return err
	}

	sourceWorkbook, err := u.loadStoredOutput(ctx, previousOutputs, orderfill.SourceOutputFileName(sourceInput.Name))
	if err != nil {
		return err
	}

	edits := make([]orderfill.ManualEdit, 0, len(message.Edits))
	for _, edit := range message.Edits {
		edits = append(edits, orderfill.ManualEdit{Key: edit.Key, Value: edit.Value, Comment: edit.Comment})
	}

	outputs := make([]port.OutputFile, 0, len(blankInputs)+1)
	for index, blankInput := range blankInputs {
		name := orderfill.BlankOutputFileName(blankInput.Name, "")
		blankWorkbook, err := u.loadStoredOutput(ctx, previousOutputs, name)
		if err != nil {
			return err
		}
		if err := orderfill.ApplyFinalEdits(orderfill.FinalizeCommand{
			Source: sourceWorkbook,
			Blank:  blankWorkbook,
			Rows:   rowsForBlank(rows, blankID(index)),
			Edits:  edits,
			Brand:  message.Brand,
		}); err != nil {
			return err
		}
		output, err := u.saveWorkbook(ctx, message.JobID, blankWorkbook, name, "Скачать заполненный бланк")
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
	}

	sourceOutput, err := u.saveWorkbook(ctx, message.JobID, sourceWorkbook, orderfill.SourceOutputFileName(sourceInput.Name), "Скачать заполненную таблицу заказа")
	if err != nil {
		return err
	}
	outputs = append(outputs, sourceOutput)

	if err := u.reports.Save(ctx, message.JobID, summary, rows, u.now()); err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	if err := u.jobs.SaveResult(ctx, message.JobID, "completed", assignOutputIDs(outputs), u.now()); err != nil {
		return fmt.Errorf("save job result: %w", err)
	}
	return nil
}

func (u *ProcessJob) loadWorkbook(ctx context.Context, key string) (spreadsheet.Workbook, error) {
	content, err := u.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read input %s: %w", key, err)
	}
	workbook, err := u.codec.Load(content)
	if err != nil {
		return nil, fmt.Errorf("%w: не удалось прочитать файл %s: %v", orderfill.ErrInvalidInput, key, err)
	}
	return workbook, nil
}

func (u *ProcessJob) loadStoredOutput(ctx context.Context, outputs []port.OutputFile, name string) (spreadsheet.Workbook, error) {
	for _, output := range outputs {
		if output.Name == name {
			return u.loadWorkbook(ctx, output.StorageKey)
		}
	}
	return nil, fmt.Errorf("generated file %q was not found for finalization", name)
}

func (u *ProcessJob) saveWorkbook(ctx context.Context, jobID string, workbook spreadsheet.Workbook, name string, label string) (port.OutputFile, error) {
	content, err := workbook.Save()
	if err != nil {
		return port.OutputFile{}, fmt.Errorf("serialize %s: %w", name, err)
	}
	key := fmt.Sprintf("jobs/%s/outputs/%s", jobID, name)
	if err := u.storage.Put(ctx, key, workbookContentType, content); err != nil {
		return port.OutputFile{}, fmt.Errorf("store %s: %w", name, err)
	}
	return port.OutputFile{
		Label:       label,
		Name:        name,
		ContentType: workbookContentType,
		SizeBytes:   int64(len(content)),
		StorageKey:  key,
	}, nil
}

// assignOutputIDs gives every generated file a short, URL-safe identifier so it
// can be addressed by the download endpoint. Storage keys contain slashes and
// Cyrillic characters and are unsuitable as path segments.
func assignOutputIDs(outputs []port.OutputFile) []port.OutputFile {
	for index := range outputs {
		outputs[index].ID = fmt.Sprintf("output-%d", index+1)
	}
	return outputs
}

func splitInputs(inputs []port.MessageFile) (port.MessageFile, []port.MessageFile, error) {
	var source port.MessageFile
	blanks := make([]port.MessageFile, 0, len(inputs))
	for _, input := range inputs {
		switch input.Role {
		case port.RoleSource:
			source = input
		case port.RoleBlank:
			blanks = append(blanks, input)
		}
	}
	if source.StorageKey == "" {
		return port.MessageFile{}, nil, fmt.Errorf("%w: не хватает таблицы заказа товара", orderfill.ErrInvalidInput)
	}
	if len(blanks) == 0 {
		return port.MessageFile{}, nil, fmt.Errorf("%w: не хватает бланка заказа", orderfill.ErrInvalidInput)
	}
	return source, blanks, nil
}

// blankID is stable across the process and finalize stages because the queue
// message keeps the blank order the user uploaded.
func blankID(index int) string {
	return fmt.Sprintf("blank-%d", index+1)
}

func rowsForBlank(rows []orderfill.ReportRow, id string) []orderfill.ReportRow {
	filtered := make([]orderfill.ReportRow, 0, len(rows))
	for _, row := range rows {
		if row.BlankID == id {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func mergeSummary(accumulated orderfill.Summary, next orderfill.Summary) orderfill.Summary {
	if accumulated.Brand == "" {
		merged := next
		return merged
	}
	accumulated.Filled += next.Filled
	accumulated.LeftBlank += next.LeftBlank
	accumulated.Suspicious += next.Suspicious
	accumulated.Unmatched += next.Unmatched
	accumulated.Duplicates += next.Duplicates
	accumulated.BlankDuplicateArticles += next.BlankDuplicateArticles
	return accumulated
}

func userMessage(err error) string {
	if errors.Is(err, orderfill.ErrInvalidInput) {
		message := err.Error()
		if index := len(orderfill.ErrInvalidInput.Error()) + 2; index < len(message) {
			return message[index:]
		}
		return message
	}
	return "Не удалось обработать файлы. Попробуйте еще раз или обратитесь в поддержку."
}

func millisSince(start time.Time, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}
