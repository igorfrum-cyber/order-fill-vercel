// Package usecase contains the worker application logic: it orchestrates the
// domain engine, object storage and the job store without knowing about Redis,
// PostgreSQL, S3 or xlsx internals.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"order-fill/services/document-service/internal/app/port"
	"order-fill/services/document-service/internal/domain/orderfill"
	"order-fill/services/document-service/internal/domain/preview"
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
	progress := newJobProgress(u.jobs, message.JobID, u.now)
	if err := u.jobs.MarkProcessing(ctx, message.JobID, u.now()); err != nil {
		return fmt.Errorf("mark job processing: %w", err)
	}
	progress.Set(ctx, 0.04, "Забираю файлы")
	if message.Type != "order_fill" {
		return fmt.Errorf("%w: тип задачи %q пока не поддерживается сервисом", orderfill.ErrInvalidInput, message.Type)
	}

	sourceInput, blankInputs, err := splitInputs(message.Inputs)
	if err != nil {
		return err
	}

	type loadedFile struct {
		workbook spreadsheet.Workbook
	}
	sourceLoaded := loadedFile{}
	blanksLoaded := make([]loadedFile, len(blankInputs))
	var sourceFrac, blankFrac float64
	var fracMu sync.Mutex
	publishLoad := func() {
		fracMu.Lock()
		src := sourceFrac
		blank := blankFrac
		fracMu.Unlock()
		messageText := "Читаю таблицу заказа"
		if src >= 0.999 && blank < 1 {
			messageText = "Читаю бланк"
		}
		progress.Set(ctx, 0.08+0.42*src+0.08*blank, messageText)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		workbook, err := u.loadWorkbook(groupCtx, sourceInput.StorageKey, func(fraction float64) {
			fracMu.Lock()
			sourceFrac = fraction
			fracMu.Unlock()
			publishLoad()
		})
		if err != nil {
			return err
		}
		sourceLoaded.workbook = workbook
		fracMu.Lock()
		sourceFrac = 1
		fracMu.Unlock()
		publishLoad()
		return nil
	})
	group.Go(func() error {
		inner, innerCtx := errgroup.WithContext(groupCtx)
		for index, blankInput := range blankInputs {
			inner.Go(func() error {
				workbook, err := u.loadWorkbook(innerCtx, blankInput.StorageKey, func(fraction float64) {
					fracMu.Lock()
					if fraction > blankFrac {
						blankFrac = fraction
					}
					fracMu.Unlock()
					publishLoad()
				})
				if err != nil {
					return err
				}
				blanksLoaded[index].workbook = workbook
				return nil
			})
		}
		if err := inner.Wait(); err != nil {
			return err
		}
		fracMu.Lock()
		blankFrac = 1
		fracMu.Unlock()
		publishLoad()
		return nil
	})
	if err := group.Wait(); err != nil {
		return err
	}

	progress.Set(ctx, 0.58, "Определяю бренд и месяц")
	detectedBrand, orderMonth, err := resolveSourceIdentity(sourceLoaded.workbook)
	if err != nil {
		return err
	}
	if err := u.jobs.SetIdentity(ctx, message.JobID, detectedBrand, orderMonth, u.now()); err != nil {
		return fmt.Errorf("save detected brand: %w", err)
	}

	blankNames := make([]string, len(blankInputs))
	for index, blankInput := range blankInputs {
		blankNames[index] = blankInput.Name
	}
	plans, err := orderfill.PlanBlanks(detectedBrand, blankNames)
	if err != nil {
		return err
	}

	outputs := make([]port.OutputFile, 0, len(blankInputs)+1)
	rows := make([]orderfill.ReportRow, 0)
	summary := orderfill.Summary{}

	for _, plan := range plans {
		progress.Set(ctx, 0.60, "Сверяю с бланком")
		result, err := orderfill.Fill(orderfill.FillCommand{
			Source:     sourceLoaded.workbook,
			Blank:      blanksLoaded[plan.Index].workbook,
			OrderMonth: orderMonth,
			Brand:      detectedBrand,
			BlankID:    plan.ID,
			BlankLabel: plan.Label,
			OnProgress: func(fraction float64, text string) {
				progress.Set(ctx, 0.60+0.22*fraction, text)
			},
		})
		if err != nil {
			return err
		}
		rows = append(rows, result.Rows...)
		summary = mergeSummary(summary, result.Summary)
	}

	progress.Set(ctx, 0.84, "Сохраняю файлы")
	saved := make([]port.OutputFile, len(blankInputs)+1)
	saveGroup, saveCtx := errgroup.WithContext(ctx)
	for index, blankInput := range blankInputs {
		saveGroup.Go(func() error {
			output, err := u.saveWorkbook(saveCtx, message.JobID, blanksLoaded[index].workbook, orderfill.BlankOutputFileName(blankInput.Name, ""), "Скачать заполненный бланк")
			if err != nil {
				return err
			}
			saved[index] = output
			return nil
		})
	}
	saveGroup.Go(func() error {
		output, err := u.saveWorkbook(saveCtx, message.JobID, sourceLoaded.workbook, orderfill.SourceOutputFileName(sourceInput.Name), "Скачать заполненную таблицу заказа")
		if err != nil {
			return err
		}
		saved[len(blankInputs)] = output
		return nil
	})
	if err := saveGroup.Wait(); err != nil {
		return err
	}
	outputs = append(outputs, assignOutputIDs(saved)...)

	progress.Set(ctx, 0.92, "Готовлю превью")
	workbooks := make([]spreadsheet.Workbook, len(blankInputs)+1)
	for index := range blankInputs {
		workbooks[index] = blanksLoaded[index].workbook
	}
	workbooks[len(blankInputs)] = sourceLoaded.workbook
	if err := u.writePreviews(ctx, message.JobID, outputs, workbooks); err != nil {
		return err
	}

	progress.Set(ctx, 0.95, "Сохраняю отчёт")
	if err := u.reports.Save(ctx, message.JobID, summary, rows, u.now()); err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	if err := u.jobs.SaveResult(ctx, message.JobID, "needs_review", outputs, u.now()); err != nil {
		return fmt.Errorf("save job result: %w", err)
	}
	return nil
}

func (u *ProcessJob) finalize(ctx context.Context, message port.JobMessage) error {
	progress := newJobProgress(u.jobs, message.JobID, u.now)
	progress.Set(ctx, 0.1, "Готовлю файлы")
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

	sourceWorkbook, err := u.loadStoredOutput(ctx, previousOutputs, orderfill.SourceOutputFileName(sourceInput.Name), nil)
	if err != nil {
		return err
	}

	edits := make([]orderfill.ManualEdit, 0, len(message.Edits))
	for _, edit := range message.Edits {
		edits = append(edits, orderfill.ManualEdit{Key: edit.Key, Value: edit.Value, Comment: edit.Comment})
	}

	outputs := make([]port.OutputFile, 0, len(blankInputs)+1)
	workbooks := make([]spreadsheet.Workbook, 0, len(blankInputs)+1)
	for index, blankInput := range blankInputs {
		progress.Set(ctx, 0.25+0.4*float64(index)/float64(max(len(blankInputs), 1)), "Вношу правки в бланк")
		name := orderfill.BlankOutputFileName(blankInput.Name, "")
		blankWorkbook, err := u.loadStoredOutput(ctx, previousOutputs, name, nil)
		if err != nil {
			return err
		}
		detectedBrand := message.Brand
		if detectedBrand == "" {
			detected, detectErr := orderfill.DetectBrand(sourceWorkbook)
			if detectErr != nil {
				return detectErr
			}
			detectedBrand = detected
		}
		if err := orderfill.ApplyFinalEdits(orderfill.FinalizeCommand{
			Source: sourceWorkbook,
			Blank:  blankWorkbook,
			Rows:   rowsForBlank(rows, blankID(index)),
			Edits:  edits,
			Brand:  detectedBrand,
		}); err != nil {
			return err
		}
		output, err := u.saveWorkbook(ctx, message.JobID, blankWorkbook, name, "Скачать заполненный бланк")
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
		workbooks = append(workbooks, blankWorkbook)
	}

	progress.Set(ctx, 0.85, "Сохраняю файлы")
	sourceOutput, err := u.saveWorkbook(ctx, message.JobID, sourceWorkbook, orderfill.SourceOutputFileName(sourceInput.Name), "Скачать заполненную таблицу заказа")
	if err != nil {
		return err
	}
	outputs = append(outputs, sourceOutput)
	workbooks = append(workbooks, sourceWorkbook)
	outputs = assignOutputIDs(outputs)

	progress.Set(ctx, 0.9, "Готовлю превью")
	if err := u.writePreviews(ctx, message.JobID, outputs, workbooks); err != nil {
		return err
	}

	if err := u.reports.Save(ctx, message.JobID, summary, rows, u.now()); err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	if err := u.jobs.SaveResult(ctx, message.JobID, "completed", outputs, u.now()); err != nil {
		return fmt.Errorf("save job result: %w", err)
	}
	return nil
}

func (u *ProcessJob) loadWorkbook(ctx context.Context, key string, report func(float64)) (spreadsheet.Workbook, error) {
	content, err := u.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read input %s: %w", key, err)
	}
	if loader, ok := u.codec.(spreadsheet.ProgressCodec); ok {
		workbook, err := loader.LoadWithProgress(content, report)
		if err != nil {
			return nil, fmt.Errorf("%w: не удалось прочитать файл %s: %v", orderfill.ErrInvalidInput, key, err)
		}
		return workbook, nil
	}
	workbook, err := u.codec.Load(content)
	if err != nil {
		return nil, fmt.Errorf("%w: не удалось прочитать файл %s: %v", orderfill.ErrInvalidInput, key, err)
	}
	if report != nil {
		report(1)
	}
	return workbook, nil
}

func (u *ProcessJob) loadStoredOutput(ctx context.Context, outputs []port.OutputFile, name string, report func(float64)) (spreadsheet.Workbook, error) {
	for _, output := range outputs {
		if output.Name == name {
			return u.loadWorkbook(ctx, output.StorageKey, report)
		}
	}
	return nil, fmt.Errorf("generated file %q was not found for finalization", name)
}

func (u *ProcessJob) writePreviews(ctx context.Context, jobID string, outputs []port.OutputFile, workbooks []spreadsheet.Workbook) error {
	if len(outputs) != len(workbooks) {
		return fmt.Errorf("preview: %d outputs and %d workbooks", len(outputs), len(workbooks))
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(4)
	for index := range outputs {
		output := outputs[index]
		workbook := workbooks[index]
		group.Go(func() error {
			return u.writePreview(groupCtx, jobID, output.ID, workbook)
		})
	}
	return group.Wait()
}

func (u *ProcessJob) writePreview(ctx context.Context, jobID string, fileID string, workbook spreadsheet.Workbook) error {
	objects, err := preview.Encode(preview.Capture(workbook))
	if err != nil {
		return fmt.Errorf("encode preview %s: %w", fileID, err)
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(8)
	for _, object := range objects {
		group.Go(func() error {
			key := fmt.Sprintf("jobs/%s/preview/%s/%s", jobID, fileID, object.Name)
			if err := u.storage.Put(groupCtx, key, object.ContentType, object.Content); err != nil {
				return fmt.Errorf("store preview %s: %w", key, err)
			}
			return nil
		})
	}
	return group.Wait()
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

func resolveSourceIdentity(workbook spreadsheet.Workbook) (string, string, error) {
	detectedBrand, err := orderfill.DetectBrand(workbook)
	if err != nil {
		return "", "", err
	}
	orderMonth, _, err := orderfill.InferOrderMonth(workbook)
	if err != nil {
		return "", "", err
	}
	return detectedBrand, orderMonth, nil
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
	accumulated.NotInBlank += next.NotInBlank
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
