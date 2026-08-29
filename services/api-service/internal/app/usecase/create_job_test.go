package usecase

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
)

type fakeRepository struct {
	created  []job.Job
	stored   map[string]job.Job
	failWith error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{stored: map[string]job.Job{}}
}

func (r *fakeRepository) Create(_ context.Context, entity job.Job) error {
	if r.failWith != nil {
		return r.failWith
	}
	r.created = append(r.created, entity)
	r.stored[entity.ID] = entity
	return nil
}

func (r *fakeRepository) Get(_ context.Context, id string) (job.Job, error) {
	entity, ok := r.stored[id]
	if !ok {
		return job.Job{}, job.ErrNotFound
	}
	return entity, nil
}

func (r *fakeRepository) UpdateStatus(_ context.Context, id string, status job.Status, updatedAt time.Time) (job.Job, error) {
	entity, ok := r.stored[id]
	if !ok {
		return job.Job{}, job.ErrNotFound
	}
	entity.Status = status
	entity.UpdatedAt = updatedAt
	r.stored[id] = entity
	return entity, nil
}

type fakeStorage struct {
	objects  map[string]port.Object
	failWith error
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{objects: map[string]port.Object{}}
}

func (s *fakeStorage) Put(_ context.Context, key string, contentType string, content []byte) error {
	if s.failWith != nil {
		return s.failWith
	}
	s.objects[key] = port.Object{Content: content, ContentType: contentType}
	return nil
}

func (s *fakeStorage) Get(_ context.Context, key string) (port.Object, error) {
	object, ok := s.objects[key]
	if !ok {
		return port.Object{}, job.ErrNotFound
	}
	return object, nil
}

type fakePublisher struct {
	messages []port.JobMessage
	failWith error
}

func (p *fakePublisher) Publish(_ context.Context, message port.JobMessage) error {
	if p.failWith != nil {
		return p.failWith
	}
	p.messages = append(p.messages, message)
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func fixedClock() port.Clock {
	moment := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	return func() time.Time { return moment }
}

func validCommand() CreateJobCommand {
	return CreateJobCommand{
		Type:       job.TypeOrderFill,
		Brand:      "angiopharm",
		OrderMonth: "2026-09",
		Uploads: []job.Upload{
			{Role: job.RoleSource, Name: "Заказ.xlsx", ContentType: "application/xlsx", Content: []byte("source")},
			{Role: job.RoleBlank, Name: "Бланк.xlsx", ContentType: "application/xlsx", Content: []byte("blank")},
		},
	}
}

func TestCreateJobStoresUploadsRecordsJobAndQueuesIt(t *testing.T) {
	repository := newFakeRepository()
	storage := newFakeStorage()
	publisher := &fakePublisher{}
	useCase := NewCreateJob(repository, storage, publisher, func() string { return "job-1" }, fixedClock(), testLogger(), nil)

	entity, err := useCase.Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entity.Status != job.StatusQueued {
		t.Fatalf("expected a queued job, got %q", entity.Status)
	}
	if len(storage.objects) != 2 {
		t.Fatalf("expected both uploads to be stored, got %d", len(storage.objects))
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("expected one queue message, got %d", len(publisher.messages))
	}
	message := publisher.messages[0]
	if message.Stage != port.StageProcess || message.JobID != "job-1" {
		t.Fatalf("unexpected message %+v", message)
	}
	if len(message.Inputs) != 2 || message.Inputs[0].Role != string(job.RoleSource) {
		t.Fatalf("unexpected message inputs %+v", message.Inputs)
	}
	if message.Inputs[0].StorageKey != entity.InputFiles[0].StorageKey {
		t.Fatal("the queue message must point at the stored objects")
	}
}

func TestCreateJobRejectsMissingBlank(t *testing.T) {
	useCase := NewCreateJob(newFakeRepository(), newFakeStorage(), &fakePublisher{}, func() string { return "job-1" }, fixedClock(), testLogger(), nil)
	command := validCommand()
	command.Uploads = command.Uploads[:1]

	if _, err := useCase.Execute(context.Background(), command); !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("expected an invalid job error, got %v", err)
	}
}

func TestCreateJobRejectsNonWorkbookUpload(t *testing.T) {
	useCase := NewCreateJob(newFakeRepository(), newFakeStorage(), &fakePublisher{}, func() string { return "job-1" }, fixedClock(), testLogger(), nil)
	command := validCommand()
	command.Uploads[1].Name = "бланк.pdf"

	if _, err := useCase.Execute(context.Background(), command); !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("expected an invalid job error, got %v", err)
	}
}

func TestCreateJobDoesNotRecordAJobWhenTheQueueIsDown(t *testing.T) {
	repository := newFakeRepository()
	publisher := &fakePublisher{failWith: errors.New("redis is down")}
	useCase := NewCreateJob(repository, newFakeStorage(), publisher, func() string { return "job-1" }, fixedClock(), testLogger(), nil)

	if _, err := useCase.Execute(context.Background(), validCommand()); err == nil {
		t.Fatal("expected the queue failure to surface")
	}
}

func TestSubmitEditsQueuesFinalizationForAReviewableJob(t *testing.T) {
	repository := newFakeRepository()
	publisher := &fakePublisher{}
	create := NewCreateJob(repository, newFakeStorage(), publisher, func() string { return "job-1" }, fixedClock(), testLogger(), nil)
	if _, err := create.Execute(context.Background(), validCommand()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := repository.UpdateStatus(context.Background(), "job-1", job.StatusNeedsReview, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	useCase := NewSubmitEdits(repository, publisher, fixedClock(), testLogger())
	entity, err := useCase.Execute(context.Background(), "job-1", []job.ManualEdit{{Key: "blank-1:7", Value: "12", Comment: "до коробки"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entity.Status != job.StatusFinalizing {
		t.Fatalf("expected the job to move to finalizing, got %q", entity.Status)
	}
	last := publisher.messages[len(publisher.messages)-1]
	if last.Stage != port.StageFinalize || len(last.Edits) != 1 {
		t.Fatalf("unexpected finalize message %+v", last)
	}
}

func TestSubmitEditsRejectsAJobThatIsStillProcessing(t *testing.T) {
	repository := newFakeRepository()
	create := NewCreateJob(repository, newFakeStorage(), &fakePublisher{}, func() string { return "job-1" }, fixedClock(), testLogger(), nil)
	if _, err := create.Execute(context.Background(), validCommand()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	useCase := NewSubmitEdits(repository, &fakePublisher{}, fixedClock(), testLogger())
	_, err := useCase.Execute(context.Background(), "job-1", nil)
	if !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("expected an invalid job error, got %v", err)
	}
}

func TestDownloadFileReturnsTheStoredWorkbook(t *testing.T) {
	repository := newFakeRepository()
	storage := newFakeStorage()
	entity, err := job.NewJob("job-1", job.TypeOrderFill, "angiopharm", "2026-09", time.Now(), []job.InputFile{{ID: "in", Role: job.RoleBlank, Name: "b.xlsx"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entity.OutputFiles = []job.OutputFile{{ID: "output-1", Name: "Бланк заполненный.xlsx", ContentType: "application/xlsx", StorageKey: "jobs/job-1/outputs/blank"}}
	if err := repository.Create(context.Background(), entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := storage.Put(context.Background(), "jobs/job-1/outputs/blank", "application/xlsx", []byte("filled")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	download, err := NewDownloadFile(repository, storage).Execute(context.Background(), "job-1", "output-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(download.Content) != "filled" || download.Name != "Бланк заполненный.xlsx" {
		t.Fatalf("unexpected download %+v", download)
	}
}

func TestDownloadArchiveZipsEveryGeneratedWorkbook(t *testing.T) {
	repository := newFakeRepository()
	storage := newFakeStorage()
	entity, err := job.NewJob("job-1", job.TypeOrderFill, "angiopharm", "2026-09", time.Now(), []job.InputFile{{ID: "in", Role: job.RoleBlank, Name: "b.xlsx"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entity.OutputFiles = []job.OutputFile{
		{ID: "output-1", Name: "Бланк заполненный.xlsx", ContentType: "application/xlsx", StorageKey: "jobs/job-1/outputs/blank"},
		{ID: "output-2", Name: "Таблица заказа.xlsx", ContentType: "application/xlsx", StorageKey: "jobs/job-1/outputs/source"},
	}
	if err := repository.Create(context.Background(), entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := storage.Put(context.Background(), "jobs/job-1/outputs/blank", "application/xlsx", []byte("blank-bytes")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := storage.Put(context.Background(), "jobs/job-1/outputs/source", "application/xlsx", []byte("source-bytes")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	download, err := NewDownloadArchive(repository, storage).Execute(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if download.Name != "angiopharm_2026-09.zip" {
		t.Fatalf("unexpected archive name %q", download.Name)
	}
	if download.ContentType != "application/zip" {
		t.Fatalf("unexpected content type %q", download.ContentType)
	}

	reader, err := zip.NewReader(bytes.NewReader(download.Content), int64(len(download.Content)))
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	got := map[string]string{}
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil {
			t.Fatalf("read entry %s: %v", file.Name, err)
		}
		got[file.Name] = string(body)
	}
	if got["Бланк заполненный.xlsx"] != "blank-bytes" || got["Таблица заказа.xlsx"] != "source-bytes" {
		t.Fatalf("unexpected archive contents %+v", got)
	}
}

func TestDownloadArchiveRequiresGeneratedFiles(t *testing.T) {
	repository := newFakeRepository()
	entity, err := job.NewJob("job-1", job.TypeOrderFill, "angiopharm", "2026-09", time.Now(), []job.InputFile{{ID: "in", Role: job.RoleBlank, Name: "b.xlsx"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repository.Create(context.Background(), entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NewDownloadArchive(repository, newFakeStorage()).Execute(context.Background(), "job-1"); !errors.Is(err, job.ErrNotFound) {
		t.Fatalf("expected a not found error, got %v", err)
	}
}
