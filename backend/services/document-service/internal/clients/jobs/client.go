package jobs

import "context"

type Report struct {
	ToOrder        int
	OrderNotNeeded int
	NeedsDecision  int
}

type Client interface {
	Complete(ctx context.Context, jobID string, report Report, fileIDs []string) error
	Fail(ctx context.Context, jobID, message string) error
}
