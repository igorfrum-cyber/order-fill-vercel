// Package queue consumes job messages published by api-service through Redis.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"order-fill/backend/services/document-service/internal/app/port"
)

// DefaultStreamName is the Redis stream both services agree on.
const DefaultStreamName = "order-fill:jobs"

const (
	DefaultGroupName    = "document-service"
	messagePayloadField = "payload"
	serviceName         = "document-service"
	defaultClaimMinIdle = 5 * time.Minute
)

// pollTimeout keeps XREADGROUP short so the loop notices a cancelled context
// promptly; errorBackoff stops a broken connection from spinning the loop.
const (
	pollTimeout  = 3 * time.Second
	errorBackoff = time.Second
)

// Handler processes one job message. Returning an error drops the message: the
// use case has already recorded the failure on the job itself.
type Handler func(ctx context.Context, message port.JobMessage) error

// Consumer reads job messages from a Redis stream consumer group.
type Consumer struct {
	client       *redis.Client
	stream       string
	group        string
	consumer     string
	claimMinIdle time.Duration
	logger       *slog.Logger
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
	return &Consumer{
		client:       redis.NewClient(options),
		stream:       stream,
		group:        DefaultGroupName,
		consumer:     consumerName(),
		claimMinIdle: defaultClaimMinIdle,
		logger:       logger,
	}, nil
}

// Run polls the queue until the context is cancelled. A single bad message never
// stops the loop.
func (c *Consumer) Run(ctx context.Context, handle Handler) error {
	if handle == nil {
		return fmt.Errorf("queue consumer requires a handler")
	}
	if ctx.Err() != nil {
		return nil
	}
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	c.logger.InfoContext(ctx, "queue consumer started",
		"service", serviceName,
		"event", "consumer_started",
		"stream", c.stream,
		"group", c.group,
		"consumer", c.consumer,
		"error_code", "",
	)

	for {
		if ctx.Err() != nil {
			c.logger.InfoContext(ctx, "queue consumer stopped",
				"service", serviceName,
				"event", "consumer_stopped",
				"stream", c.stream,
				"group", c.group,
				"consumer", c.consumer,
				"error_code", "",
			)
			return nil
		}

		message, ok, err := c.next(ctx)
		if err != nil {
			if ok {
				c.logger.ErrorContext(ctx, "queue message rejected",
					"service", serviceName,
					"job_id", "",
					"event", "message_rejected",
					"error_code", "invalid_payload",
					"message_id", message.id,
					"error", err,
				)
				if ackErr := c.ack(ctx, message.id); ackErr != nil && ctx.Err() == nil {
					c.logger.ErrorContext(ctx, "queue ack failed",
						"service", serviceName,
						"job_id", "",
						"event", "queue_ack_failed",
						"error_code", "queue_unavailable",
						"message_id", message.id,
						"error", ackErr,
					)
				}
				continue
			}
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
		if !ok {
			continue
		}
		c.dispatch(ctx, message.payload, handle)
		if err := c.ack(ctx, message.id); err != nil {
			if ctx.Err() != nil {
				continue
			}
			c.logger.ErrorContext(ctx, "queue ack failed",
				"service", serviceName,
				"job_id", "",
				"event", "queue_ack_failed",
				"error_code", "queue_unavailable",
				"message_id", message.id,
				"error", err,
			)
			wait(ctx, errorBackoff)
		}
	}
}

func (c *Consumer) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close queue client: %w", err)
	}
	return nil
}

type queuedMessage struct {
	id      string
	payload string
}

func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err != nil {
		if isBusyGroupError(err) {
			return nil
		}
		return fmt.Errorf("create consumer group %s for stream %s: %w", c.group, c.stream, err)
	}
	return nil
}

func (c *Consumer) next(ctx context.Context) (queuedMessage, bool, error) {
	if message, ok, err := c.claim(ctx); err != nil || ok {
		return message, ok, err
	}
	return c.poll(ctx)
}

func (c *Consumer) claim(ctx context.Context) (queuedMessage, bool, error) {
	messages, _, err := c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   c.stream,
		Group:    c.group,
		Consumer: c.consumer,
		MinIdle:  c.claimMinIdle,
		Start:    "0-0",
		Count:    1,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return queuedMessage{}, false, nil
		}
		return queuedMessage{}, false, fmt.Errorf("claim pending job from %s/%s: %w", c.stream, c.group, err)
	}
	return firstStreamMessage(messages)
}

// poll returns ok=false when the block timeout expired with no message.
func (c *Consumer) poll(ctx context.Context) (queuedMessage, bool, error) {
	streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.group,
		Consumer: c.consumer,
		Streams:  []string{c.stream, ">"},
		Count:    1,
		Block:    pollTimeout,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return queuedMessage{}, false, nil
		}
		return queuedMessage{}, false, fmt.Errorf("read job from %s/%s: %w", c.stream, c.group, err)
	}
	if len(streams) == 0 {
		return queuedMessage{}, false, nil
	}
	return firstStreamMessage(streams[0].Messages)
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

func (c *Consumer) ack(ctx context.Context, messageID string) error {
	if messageID == "" {
		return nil
	}
	if err := c.client.XAck(ctx, c.stream, c.group, messageID).Err(); err != nil {
		return fmt.Errorf("ack job message %s from %s/%s: %w", messageID, c.stream, c.group, err)
	}
	return nil
}

func firstStreamMessage(messages []redis.XMessage) (queuedMessage, bool, error) {
	if len(messages) == 0 {
		return queuedMessage{}, false, nil
	}
	payload, err := streamMessagePayload(messages[0])
	if err != nil {
		return queuedMessage{id: messages[0].ID}, true, err
	}
	return queuedMessage{id: messages[0].ID, payload: payload}, true, nil
}

func streamMessagePayload(message redis.XMessage) (string, error) {
	value, ok := message.Values[messagePayloadField]
	if !ok {
		return "", fmt.Errorf("stream message %s missing %q field", message.ID, messagePayloadField)
	}
	switch payload := value.(type) {
	case string:
		return payload, nil
	case []byte:
		return string(payload), nil
	default:
		return "", fmt.Errorf("stream message %s %q field has type %T", message.ID, messagePayloadField, value)
	}
}

func decodeMessage(payload []byte) (port.JobMessage, error) {
	var message port.JobMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return port.JobMessage{}, fmt.Errorf("unmarshal job message: %w", err)
	}
	return message, nil
}

func consumerName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

func isBusyGroupError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(err.Error()), "BUSYGROUP")
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
