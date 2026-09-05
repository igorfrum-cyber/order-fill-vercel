package jobs

import (
	"context"

	"order-fill/backend/services/job-service/internal/domain"
)

func (s *Service) Complete(ctx context.Context, jobID string, report domain.Report, files []domain.FileRef) error {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return domain.ErrNotFound
	}
	if job.Status == domain.StatusFinalizing || job.Status == domain.StatusCompleted {
		job.Status = domain.StatusCompleted
	} else {
		job.Status = domain.StatusNeedsReview
	}
	if files != nil {
		job.OutputFiles = files
	}
	job.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, job); err != nil {
		return err
	}
	return s.store.SaveReport(ctx, jobID, report)
}

func (s *Service) Fail(ctx context.Context, jobID, message string) error {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return domain.ErrNotFound
	}
	job.Fail(message, s.now())
	return s.store.Update(ctx, job)
}

func (s *Service) GetReport(ctx context.Context, actor domain.Actor, jobID string) (domain.Report, error) {
	if _, err := s.Get(ctx, actor, jobID); err != nil {
		return domain.Report{}, err
	}
	return s.store.GetReport(ctx, jobID)
}

func (s *Service) UpdateProgress(ctx context.Context, jobID string, status domain.Status, message string, progress float64) error {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return domain.ErrNotFound
	}
	if status != "" {
		job.Status = status
	}
	job.Progress = progress
	job.ProgressMessage = message
	job.UpdatedAt = s.now().UTC()
	return s.store.Update(ctx, job)
}
