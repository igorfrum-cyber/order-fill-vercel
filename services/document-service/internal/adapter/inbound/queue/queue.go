// Package queue consumes job messages published by api-service through Redis.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"order-fill/services/document-service/internal/app/port"
)

// DefaultStreamName is the Redis list both services agree on: api-service
// LPUSHes, document-service BRPOPs, which makes the list a FIFO queue.
const DefaultStreamName = "order-fill:jobs"

const serviceName = "document-service"

// pollTimeout keeps BRPOP short so the loop notices a cancelled context
// promptly; errorBackoff stops a broken connection from spinning the loop.
const (
	pollTimeout  = 3 * time.Second
	errorBackoff = time.Second
)

// Handler processes one job message. Returning an error drops the message: the
// use case has already recorded the failure on the job itself.
type Handler func(ctx context.Context, message port.JobMessage) error

// Consumer reads job messages off a Redis list.
type Consumer struct {
	client *redis.Client
	stream string
	logger *slog.Logger
}

// NewConsumer parses a QUEUE_URL like redis://redis:6379/0. An empty stream
// falls back to DefaultStreamName.
func NewConsumer(queueURL string, stream string, logger *slog.Logger) (*Consumer, error) {
	options, err := redis.ParseURL(queueURL)
	if err != nil {
		return nil, fmt.Errorf("parse queue url: %w", err)
	}
	if strings.TrimSpace(stream) == "" {
		stream = DefaultStreamName
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{client: redis.NewClient(options), stream: stream, logger: logger}, nil
}

// Run polls the queue until the context is cancelled. A single bad message never
// stops the loop.
func (c *Consumer) Run(ctx context.Context, handle Handler) error {
	if handle == nil {
		return fmt.Errorf("queue consumer requires a handler")
	}
	c.logger.InfoContext(ctx, "queue consumer started",
		"service", serviceName,
		"event", "consumer_started",
		"stream", c.stream,
		"error_code", "",
	)

	for {
		if ctx.Err() != nil {
			c.logger.InfoContext(ctx, "queue consumer stopped",
				"service", serviceName,
				"event", "consumer_stopped",
				"stream", c.stream,
				"error_code", "",
			)
			return nil
		}

		payload, err := c.poll(ctx)
		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			c.logger.ErrorContext(ctx, "queue poll failed",
				"service", serviceName,
				"job_id", "",
				"event", "queue_poll_failed",
				"error_code", "queue_unavailable",
				"error", err,
			)
			wait(ctx, errorBackoff)
			continue
		}
		if payload == "" {
			continue
		}
		c.dispatch(ctx, payload, handle)
	}
}

func (c *Consumer) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close queue client: %w", err)
	}
	return nil
}

// poll returns an empty payload when the block timeout expired with no message.
func (c *Consumer) poll(ctx context.Context) (string, error) {
	values, err := c.client.BRPop(ctx, pollTimeout, c.stream).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", fmt.Errorf("pop job from %s: %w", c.stream, err)
	}
	if len(values) < 2 {
		return "", nil
	}
	return values[1], nil
}

func (c *Consumer) dispatch(ctx context.Context, payload string, handle Handler) {
	message, err := decodeMessage([]byte(payload))
	if err != nil {
		c.logger.ErrorContext(ctx, "queue message rejected",
			"service", serviceName,
			"job_id", "",
			"event", "message_rejected",
			"error_code", "invalid_payload",
			"error", err,
		)
		return
	}
	if err := handle(ctx, message); err != nil {
		c.logger.ErrorContext(ctx, "queue message handling failed",
			"service", serviceName,
			"job_id", message.JobID,
			"event", "message_failed",
			"error_code", "handler_error",
			"error", err,
		)
	}
}

func decodeMessage(payload []byte) (port.JobMessage, error) {
	var message port.JobMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return port.JobMessage{}, fmt.Errorf("unmarshal job message: %w", err)
	}
	return message, nil
}

// wait sleeps for the delay unless the context is cancelled first.
func wait(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
