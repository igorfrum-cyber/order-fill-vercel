package jobs

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"
)

var ErrInvalidJob = errors.New("invalid job")

type IDGenerator func() string

type Clock func() time.Time

type Repository interface {
	Save(ctx context.Context, job Job) error
	Find(ctx context.Context, id string) (Job, error)
	UpdateStatus(ctx context.Context, id string, status JobStatus) (Job, error)
	SaveReport(ctx context.Context, report Report) error
	Report(ctx context.Context, jobID string) (Report, error)
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, file UploadFile) (StoredFile, error)
}

type Queue interface {
	Enqueue(ctx context.Context, message JobMessage) error
}

type ServiceConfig struct {
	Repository Repository
	Storage    ObjectStorage
	Queue      Queue
	NewID      IDGenerator
	Now        Clock
	Logger     *slog.Logger
	Metrics    Metrics
}

type Service struct {
	repository Repository
	storage    ObjectStorage
	queue      Queue
	newID      IDGenerator
	now        Clock
	logger     *slog.Logger
	metrics    Metrics
}

type Metrics interface {
	AddJobCreated()
	AddJobFailed()
	AddBytesStored(value int64)
}

type CreateJobCommand struct {
	Type       JobType
	Brand      string
	OrderMonth string
	Files      []UploadFile
}

type UploadFile struct {
	Name        string
	ContentType string
	SizeBytes   int64
	Reader      io.Reader
}

func NewService(config ServiceConfig) *Service {
	return &Service{
		repository: config.Repository,
		storage:    config.Storage,
		queue:      config.Queue,
		newID:      defaultID(config.NewID),
		now:        defaultClock(config.Now),
		logger:     defaultLogger(config.Logger),
		metrics:    config.Metrics,
	}
}

func (s *Service) CreateJob(ctx context.Context, command CreateJobCommand) (Job, error) {
	startedAt := s.now()
	if err := validateCreateJobCommand(command); err != nil {
		s.observeFailure(ctx, "", "create_job_invalid", startedAt, "invalid_job", err)
		return Job{}, err
	}

	id := s.newID()
	inputFiles := make([]StoredFile, 0, len(command.Files))
	for _, file := range command.Files {
		key := fmt.Sprintf("jobs/%s/inputs/%s", id, cleanFileName(file.Name))
		stored, err := s.storage.Put(ctx, key, file)
		if err != nil {
			s.observeFailure(ctx, id, "store_input_failed", startedAt, "storage_error", err)
			return Job{}, fmt.Errorf("store input file: %w", err)
		}
		if s.metrics != nil {
			s.metrics.AddBytesStored(stored.SizeBytes)
		}
		inputFiles = append(inputFiles, stored)
	}

	now := s.now().UTC()
	job := Job{
		ID:          id,
		Type:        command.Type,
		Status:      JobStatusQueued,
		Brand:       strings.TrimSpace(command.Brand),
		OrderMonth:  strings.TrimSpace(command.OrderMonth),
		CreatedAt:   now,
		UpdatedAt:   now,
		InputFiles:  inputFiles,
		OutputFiles: []OutputFile{},
	}
	if err := s.repository.Save(ctx, job); err != nil {
		s.observeFailure(ctx, id, "save_job_failed", startedAt, "repository_error", err)
		return Job{}, fmt.Errorf("save job: %w", err)
	}
	if err := s.queue.Enqueue(ctx, JobMessage{JobID: job.ID, Type: job.Type}); err != nil {
		s.observeFailure(ctx, id, "enqueue_job_failed", startedAt, "queue_error", err)
		return Job{}, fmt.Errorf("enqueue job: %w", err)
	}
	if s.metrics != nil {
		s.metrics.AddJobCreated()
	}
	s.logger.InfoContext(ctx, "job created",
		"service", "api-service",
		"job_id", job.ID,
		"event", "job_created",
		"duration_ms", durationMillis(startedAt, s.now()),
		"error_code", "",
		"type", job.Type,
		"file_count", len(job.InputFiles),
	)
	return job, nil
}

func (s *Service) Find(ctx context.Context, id string) (Job, error) {
	return s.repository.Find(ctx, id)
}

func (s *Service) Report(ctx context.Context, jobID string) (Report, error) {
	return s.repository.Report(ctx, jobID)
}

func (s *Service) SubmitEdits(ctx context.Context, jobID string, edits []ManualEdit) (Job, error) {
	if strings.TrimSpace(jobID) == "" {
		return Job{}, fmt.Errorf("%w: job id is required", ErrInvalidJob)
	}
	for _, edit := range edits {
		if strings.TrimSpace(edit.Key) == "" {
			return Job{}, fmt.Errorf("%w: edit key is required", ErrInvalidJob)
		}
	}
	return s.repository.UpdateStatus(ctx, jobID, JobStatusFinalizing)
}

func (s *Service) Files(ctx context.Context, jobID string) ([]OutputFile, error) {
	job, err := s.repository.Find(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return append([]OutputFile(nil), job.OutputFiles...), nil
}

func validateCreateJobCommand(command CreateJobCommand) error {
	if command.Type != JobTypeOrderFill && command.Type != JobTypeNorthMerge {
		return fmt.Errorf("%w: unsupported job type %q", ErrInvalidJob, command.Type)
	}
	if strings.TrimSpace(command.Brand) == "" {
		return fmt.Errorf("%w: brand is required", ErrInvalidJob)
	}
	if command.Type == JobTypeOrderFill && strings.TrimSpace(command.OrderMonth) == "" {
		return fmt.Errorf("%w: order_month is required", ErrInvalidJob)
	}
	if len(command.Files) == 0 {
		return fmt.Errorf("%w: at least one file is required", ErrInvalidJob)
	}
	for _, file := range command.Files {
		if strings.TrimSpace(file.Name) == "" {
			return fmt.Errorf("%w: file name is required", ErrInvalidJob)
		}
		if file.Reader == nil {
			return fmt.Errorf("%w: file reader is required", ErrInvalidJob)
		}
	}
	return nil
}

func cleanFileName(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	return path.Base(normalized)
}

func defaultID(generator IDGenerator) IDGenerator {
	if generator != nil {
		return generator
	}
	return NewUUID
}

func defaultClock(clock Clock) Clock {
	if clock != nil {
		return clock
	}
	return time.Now
}

func defaultLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}

func (s *Service) observeFailure(ctx context.Context, jobID string, event string, startedAt time.Time, errorCode string, err error) {
	if s.metrics != nil {
		s.metrics.AddJobFailed()
	}
	s.logger.ErrorContext(ctx, "job operation failed",
		"service", "api-service",
		"job_id", jobID,
		"event", event,
		"duration_ms", durationMillis(startedAt, s.now()),
		"error_code", errorCode,
		"error", err,
	)
}

func durationMillis(start time.Time, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func NewUUID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
