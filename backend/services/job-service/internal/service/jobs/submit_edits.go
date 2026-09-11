package jobs

import (
	"context"

	"order-fill/backend/services/job-service/internal/domain"
	"order-fill/backend/services/job-service/internal/queue"
)

func (s *Service) SubmitEdits(ctx context.Context, actor domain.Actor, jobID string, edits []domain.Edit) (domain.Job, error) {
	job, err := s.Get(ctx, actor, jobID)
	if err != nil {
		return domain.Job{}, err
	}
	if !job.CanAcceptEdits() {
		return domain.Job{}, domain.ErrConflict
	}
	from := job.Status
	now := s.now().UTC()
	if err := s.store.CompareAndSwapStatus(ctx, job.ID, from, domain.StatusFinalizing, now); err != nil {
		return domain.Job{}, err
	}
	job.Status = domain.StatusFinalizing
	job.UpdatedAt = now
	queueEdits := make([]queue.Edit, 0, len(edits))
	for _, edit := range edits {
		queueEdits = append(queueEdits, queue.Edit{Key: edit.RowKey, Value: edit.Value, Comment: edit.Comment})
	}
	if err := s.publisher.Publish(ctx, queue.Message{
		Version:      queue.Version,
		JobID:        job.ID,
		Type:         string(job.Type),
		Stage:        "finalize",
		MatchingMode: string(job.MatchingMode),
		CompanyID:    job.CompanyID,
		Inputs:       queueInputs(append(append([]domain.FileRef{}, job.InputFiles...), job.OutputFiles...)),
		Edits:        queueEdits,
	}); err != nil {
		_ = s.store.CompareAndSwapStatus(ctx, job.ID, domain.StatusFinalizing, from, s.now().UTC())
		return domain.Job{}, err
	}
	return job, nil
}
