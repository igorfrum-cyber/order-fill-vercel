package main

import (
	"context"
	"testing"
	"time"

	"order-fill/services/document-service/internal/adapter/inbound/queue"
	"order-fill/services/document-service/internal/app/port"
)

func TestRunConsumerPoolStartsConfiguredWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 3)
	runner := consumerRunnerFunc(func(ctx context.Context, _ queue.Handler) error {
		started <- struct{}{}
		<-ctx.Done()
		return nil
	})

	errs := make(chan error, 1)
	go func() {
		errs <- runConsumerPool(ctx, runner, 3, func(context.Context, port.JobMessage) error { return nil })
	}()

	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("runConsumerPool(ctx, runner, 3, handle) started %d workers, want %d", i, 3)
		}
	}
	cancel()
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("runConsumerPool(ctx, runner, 3, handle) error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runConsumerPool(ctx, runner, 3, handle) did not stop after context cancellation")
	}
}

type consumerRunnerFunc func(context.Context, queue.Handler) error

func (f consumerRunnerFunc) Run(ctx context.Context, handle queue.Handler) error {
	return f(ctx, handle)
}
