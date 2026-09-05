package jobs

import (
	"context"
	"fmt"

	"order-fill/backend/services/job-service/internal/domain"
	"order-fill/backend/services/job-service/internal/queue"
)

func (s *Service) Create(ctx context.Context, actor domain.Actor, jobType domain.Type, fileIDs []string, brand string) (domain.Job, error) {
	if !domain.CanCreateJob(actor) {
		return domain.Job{}, domain.ErrUnauthorized
	}
	files, err := s.files.Describe(ctx, fileIDs)
	if err != nil {
		return domain.Job{}, err
	}
	uploads := make([]domain.UploadMeta, 0, len(files))
	for _, file := range files {
		uploads = append(uploads, domain.UploadMeta{Role: domain.Role(file.Kind), Name: file.Name})
	}
	if err := domain.ValidateUploads(jobType, uploads); err != nil {
		return domain.Job{}, err
	}
	mode := domain.MatchingModeStandard
	if s.companies != nil {
		got, err := s.companies.MatchingMode(ctx, actor)
		if err != nil {
			return domain.Job{}, err
		}
		mode = got
	}
	id, err := newID()
	if err != nil {
		return domain.Job{}, err
	}
	for i := range files {
		files[i].JobID = id
	}
	job, err := domain.NewJob(id, jobType, actor.UserID, actor.CompanyID, mode, s.now(), files)
	if err != nil {
		return domain.Job{}, err
	}
	if err := s.store.Create(ctx, job); err != nil {
		return domain.Job{}, err
	}
	if err := s.publisher.Publish(queue.Message{
		Version:      queue.Version,
		JobID:        job.ID,
		Type:         string(job.Type),
		Stage:        "process",
		MatchingMode: string(job.MatchingMode),
		Brand:        brand,
		Inputs:       queueInputs(files),
	}); err != nil {
		return domain.Job{}, fmt.Errorf("enqueue job: %w", err)
	}
	return job, nil
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (domain.Job, error) {
	job, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Job{}, domain.ErrNotFound
	}
	if !domain.CanAccessJob(actor, job) {
		return domain.Job{}, domain.ErrNotFound
	}
	return job, nil
}

func (s *Service) List(ctx context.Context, actor domain.Actor) ([]domain.Job, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Job, 0)
	for _, job := range items {
		if domain.CanAccessJob(actor, job) {
			out = append(out, job)
		}
	}
	return out, nil
}

func (s *Service) ListFiles(ctx context.Context, actor domain.Actor, id string) ([]domain.FileRef, error) {
	job, err := s.Get(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	out := append([]domain.FileRef{}, job.InputFiles...)
	return append(out, job.OutputFiles...), nil
}

func queueInputs(files []domain.FileRef) []queue.Input {
	out := make([]queue.Input, 0, len(files))
	for _, file := range files {
		out = append(out, queue.Input{Role: file.Kind, Name: file.Name, StorageKey: file.ObjectKey})
	}
	return out
}
