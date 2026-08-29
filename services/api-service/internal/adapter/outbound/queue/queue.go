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

// DefaultStreamName is the Redis list both services agree on: api-service
// LPUSHes, document-service BRPOPs, which makes the list a FIFO queue.
const DefaultStreamName = "order-fill:jobs"

var _ port.JobPublisher = (*Publisher)(nil)

// Publisher publishes job messages onto a Redis list.
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
	payload, err := encodeMessage(message)
	if err != nil {
		return err
	}
	if err := p.client.LPush(ctx, p.stream, payload).Err(); err != nil {
		return fmt.Errorf("push job %s onto %s: %w", message.JobID, p.stream, err)
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
