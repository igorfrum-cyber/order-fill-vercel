// Package usecase contains the api-service application logic. Each type is a
// single interactor that depends only on the ports it actually uses.
package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
)

// CreateJobCommand is the transport-independent input of CreateJob.
type CreateJobCommand struct {
	Type       job.Type
	Brand      string
	OrderMonth string
	Uploads    []job.Upload
}

// CreateJob stores the uploaded workbooks, records the job and queues it.
type CreateJob struct {
	repository JobWriter
	storage    port.ObjectStore
	publisher  port.JobPublisher
	newID      port.IDGenerator
	now        port.Clock
	logger     *slog.Logger
	metrics    port.Metrics
}

// JobWriter is the narrow slice of the repository CreateJob needs.
type JobWriter interface {
	Create(ctx context.Context, entity job.Job) error
}

func NewCreateJob(
	repository JobWriter,
	storage port.ObjectStore,
	publisher port.JobPublisher,
	newID port.IDGenerator,
	now port.Clock,
	logger *slog.Logger,
	metrics port.Metrics,
) *CreateJob {
	return &CreateJob{
		repository: repository,
		storage:    storage,
		publisher:  publisher,
		newID:      newID,
		now:        now,
		logger:     logger,
		metrics:    metrics,
	}
}

func (u *CreateJob) Execute(ctx context.Context, command CreateJobCommand) (job.Job, error) {
	startedAt := u.now()
	if err := job.ValidateUploads(command.Type, command.Uploads); err != nil {
		u.observeFailure(ctx, "", "create_job_invalid", startedAt, "invalid_job", err)
		return job.Job{}, err
	}

	id := u.newID()
	inputFiles := make([]job.InputFile, 0, len(command.Uploads))
	messageFiles := make([]port.MessageFile, 0, len(command.Uploads))
	for index, upload := range command.Uploads {
		key := job.StorageKey(id, index, upload.Name)
		if err := u.storage.Put(ctx, key, upload.ContentType, upload.Content); err != nil {
			u.observeFailure(ctx, id, "store_input_failed", startedAt, "storage_error", err)
			return job.Job{}, fmt.Errorf("store input file: %w", err)
		}
		if u.metrics != nil {
			u.metrics.AddBytesStored(int64(len(upload.Content)))
		}
		inputFiles = append(inputFiles, job.InputFile{
			ID:          key,
			Role:        upload.Role,
			Name:        job.SafeFileName(upload.Name),
			ContentType: upload.ContentType,
			SizeBytes:   int64(len(upload.Content)),
			StorageKey:  key,
		})
		messageFiles = append(messageFiles, port.MessageFile{
			Role:       string(upload.Role),
			Name:       job.SafeFileName(upload.Name),
			StorageKey: key,
		})
	}

	entity, err := job.NewJob(id, command.Type, command.Brand, command.OrderMonth, u.now(), inputFiles)
	if err != nil {
		u.observeFailure(ctx, id, "create_job_invalid", startedAt, "invalid_job", err)
		return job.Job{}, err
	}
	if err := u.repository.Create(ctx, entity); err != nil {
		u.observeFailure(ctx, id, "save_job_failed", startedAt, "repository_error", err)
		return job.Job{}, fmt.Errorf("save job: %w", err)
	}

	message := port.JobMessage{
		JobID:      entity.ID,
		Type:       string(entity.Type),
		Stage:      port.StageProcess,
		Brand:      entity.Brand,
		OrderMonth: entity.OrderMonth,
		Inputs:     messageFiles,
	}
	if err := u.publisher.Publish(ctx, message); err != nil {
		u.observeFailure(ctx, id, "enqueue_job_failed", startedAt, "queue_error", err)
		return job.Job{}, fmt.Errorf("enqueue job: %w", err)
	}
	if u.metrics != nil {
		u.metrics.AddJobCreated()
	}

	u.logger.InfoContext(ctx, "job created",
		"service", "api-service",
		"job_id", entity.ID,
		"event", "job_created",
		"duration_ms", durationMillis(startedAt, u.now()),
		"error_code", "",
		"type", entity.Type,
		"file_count", len(entity.InputFiles),
	)
	return entity, nil
}

func (u *CreateJob) observeFailure(ctx context.Context, jobID string, event string, startedAt time.Time, code string, err error) {
	if u.metrics != nil {
		u.metrics.AddJobFailed()
	}
	u.logger.ErrorContext(ctx, "job operation failed",
		"service", "api-service",
		"job_id", jobID,
		"event", event,
		"duration_ms", durationMillis(startedAt, u.now()),
		"error_code", code,
		"error", err,
	)
}

func durationMillis(start time.Time, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}
