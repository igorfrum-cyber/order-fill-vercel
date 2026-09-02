package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
)

// SubmitEdits forwards reviewer corrections to document-service and moves the
// job into finalization.
type SubmitEdits struct {
	repository port.JobRepository
	publisher  port.JobPublisher
	now        port.Clock
	logger     *slog.Logger
}

func NewSubmitEdits(
	repository port.JobRepository,
	publisher port.JobPublisher,
	now port.Clock,
	logger *slog.Logger,
) *SubmitEdits {
	return &SubmitEdits{repository: repository, publisher: publisher, now: now, logger: logger}
}

func (u *SubmitEdits) Execute(ctx context.Context, jobID string, edits []job.ManualEdit) (job.Job, error) {
	if jobID == "" {
		return job.Job{}, fmt.Errorf("%w: job id is required", job.ErrInvalid)
	}
	if err := job.ValidateEdits(edits); err != nil {
		return job.Job{}, err
	}

	entity, err := u.repository.Get(ctx, jobID)
	if err != nil {
		return job.Job{}, err
	}
	if !entity.CanAcceptEdits() {
		return job.Job{}, fmt.Errorf("%w: job is %s and cannot accept edits", job.ErrInvalid, entity.Status)
	}

	messageEdits := make([]port.MessageEdit, 0, len(edits))
	for _, edit := range edits {
		messageEdits = append(messageEdits, port.MessageEdit{Key: edit.Key, Value: edit.Value, Comment: edit.Comment})
	}
	inputs := make([]port.MessageFile, 0, len(entity.InputFiles))
	for _, file := range entity.InputFiles {
		inputs = append(inputs, port.MessageFile{Role: string(file.Role), Name: file.Name, StorageKey: file.StorageKey})
	}

	updated, err := u.repository.UpdateStatus(ctx, jobID, job.StatusFinalizing, u.now())
	if err != nil {
		return job.Job{}, err
	}

	message := port.JobMessage{
		JobID:      entity.ID,
		Type:       string(entity.Type),
		Stage:      port.StageFinalize,
		Brand:      entity.Brand,
		OrderMonth: entity.OrderMonth,
		Inputs:     inputs,
		Edits:      messageEdits,
	}
	if err := u.publisher.Publish(ctx, message); err != nil {
		return job.Job{}, fmt.Errorf("enqueue finalize job: %w", err)
	}

	u.logger.InfoContext(ctx, "job edits submitted",
		"service", "api-service",
		"job_id", jobID,
		"event", "job_edits_submitted",
		"duration_ms", 0,
		"error_code", "",
		"edit_count", len(edits),
	)
	return updated, nil
}
