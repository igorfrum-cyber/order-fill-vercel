// Package queue hands jobs over to document-service through Redis.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"order-fill/services/api-service/internal/app/port"
)

// DefaultStreamName is the Redis stream both services agree on.
const DefaultStreamName = "order-fill:jobs"

const messagePayloadField = "payload"

var _ port.JobPublisher = (*Publisher)(nil)

// Publisher publishes job messages onto a Redis stream.
type Publisher struct {
	client *redis.Client
	stream string
}

// NewPublisher parses a QUEUE_URL like redis://redis:6379/0. An empty stream
// falls back to DefaultStreamName.
func NewPublisher(queueURL string, stream string) (*Publisher, error) {
	options, err := redis.ParseURL(queueURL)
	if err != nil {
		return nil, fmt.Errorf("parse queue url: %w", err)
	}
	if strings.TrimSpace(stream) == "" {
		stream = DefaultStreamName
	}
	return &Publisher{client: redis.NewClient(options), stream: stream}, nil
}

func (p *Publisher) Publish(ctx context.Context, message port.JobMessage) error {
	values, err := streamValues(message)
	if err != nil {
		return err
	}
	if err := p.client.XAdd(ctx, &redis.XAddArgs{Stream: p.stream, Values: values}).Err(); err != nil {
		return fmt.Errorf("publish job %s to stream %s: %w", message.JobID, p.stream, err)
	}
	return nil
}

func (p *Publisher) Ping(ctx context.Context) error {
	if err := p.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping queue: %w", err)
	}
	return nil
}

func (p *Publisher) Close() error {
	if err := p.client.Close(); err != nil {
		return fmt.Errorf("close queue client: %w", err)
	}
	return nil
}

func encodeMessage(message port.JobMessage) ([]byte, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal job message %s: %w", message.JobID, err)
	}
	return payload, nil
}

func streamValues(message port.JobMessage) (map[string]any, error) {
	payload, err := encodeMessage(message)
	if err != nil {
		return nil, err
	}
	return map[string]any{messagePayloadField: string(payload)}, nil
}
