package jobs

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestServiceCreatesQueuedJobWithStoredInputs(t *testing.T) {
	repository := NewMemoryRepository()
	storage := &recordingStorage{}
	queue := &recordingQueue{}
	service := NewService(ServiceConfig{
		Repository: repository,
		Storage:    storage,
		Queue:      queue,
		NewID:      fixedID("job-1"),
		Now:        fixedNow(time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)),
	})

	job, err := service.CreateJob(context.Background(), CreateJobCommand{
		Type:       JobTypeOrderFill,
		Brand:      "angiopharm",
		OrderMonth: "2026-08",
		Files: []UploadFile{
			{
				Name:        "source.xlsx",
				ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				SizeBytes:   6,
				Reader:      bytes.NewBufferString("source"),
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}

	if job.ID != "job-1" || job.Status != JobStatusQueued {
		t.Fatalf("expected queued job-1, got id=%q status=%q", job.ID, job.Status)
	}
	if len(job.InputFiles) != 1 || job.InputFiles[0].Name != "source.xlsx" {
		t.Fatalf("expected stored input metadata, got %#v", job.InputFiles)
	}
	if got := storage.keys[0]; got != "jobs/job-1/inputs/source.xlsx" {
		t.Fatalf("unexpected storage key %q", got)
	}
	if len(queue.messages) != 1 || queue.messages[0].JobID != "job-1" {
		t.Fatalf("expected one queued job message, got %#v", queue.messages)
	}

	saved, err := repository.Find(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if saved.ID != job.ID {
		t.Fatalf("expected saved job %q, got %q", job.ID, saved.ID)
	}
}

func TestServicePreviewProcessorMovesJobToReviewAndStoresReport(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(ServiceConfig{
		Repository:       repository,
		Storage:          &recordingStorage{},
		Queue:            &recordingQueue{},
		NewID:            fixedID("job-1"),
		Now:              fixedNow(time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)),
		PreviewProcessor: StaticPreviewProcessor{},
	})

	job, err := service.CreateJob(context.Background(), CreateJobCommand{
		Type:       JobTypeOrderFill,
		Brand:      "angiopharm",
		OrderMonth: "2026-08",
		Files: []UploadFile{{
			Name:        "source.xlsx",
			ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			Reader:      bytes.NewBufferString("source"),
		}},
	})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}

	if job.Status != JobStatusNeedsReview {
		t.Fatalf("expected needs_review preview job, got %q", job.Status)
	}
	report, err := service.Report(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("expected preview report row, got %#v", report.Rows)
	}

	completed, err := service.SubmitEdits(context.Background(), job.ID, []ManualEdit{{Key: "preview:job-1", Value: "1"}})
	if err != nil {
		t.Fatalf("SubmitEdits returned error: %v", err)
	}
	if completed.Status != JobStatusCompleted {
		t.Fatalf("expected completed preview job, got %q", completed.Status)
	}
	files, err := service.Files(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("Files returned error: %v", err)
	}
	if len(files) != 1 || files[0].DownloadURL == "" {
		t.Fatalf("expected preview output file, got %#v", files)
	}
}

func fixedID(value string) IDGenerator {
	return func() string {
		return value
	}
}

func fixedNow(value time.Time) Clock {
	return func() time.Time {
		return value
	}
}

type recordingStorage struct {
	keys []string
}

func (s *recordingStorage) Put(_ context.Context, key string, file UploadFile) (StoredFile, error) {
	content, err := io.ReadAll(file.Reader)
	if err != nil {
		return StoredFile{}, err
	}
	s.keys = append(s.keys, key)
	return StoredFile{
		ID:          "stored-" + file.Name,
		Name:        file.Name,
		ContentType: file.ContentType,
		SizeBytes:   int64(len(content)),
		Key:         key,
	}, nil
}

type recordingQueue struct {
	messages []JobMessage
}

func (q *recordingQueue) Enqueue(_ context.Context, message JobMessage) error {
	q.messages = append(q.messages, message)
	return nil
}
