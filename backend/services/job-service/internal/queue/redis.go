package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

const Version = "v1"

type Input struct {
	Role       string `json:"role"`
	Name       string `json:"name"`
	StorageKey string `json:"storage_key"`
}

type Edit struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Comment string `json:"comment,omitempty"`
}

type Message struct {
	Version      string  `json:"version"`
	JobID        string  `json:"job_id"`
	Type         string  `json:"type"`
	Stage        string  `json:"stage"`
	Brand        string  `json:"brand,omitempty"`
	MatchingMode string  `json:"matching_mode"`
	Inputs       []Input `json:"inputs"`
	Edits        []Edit  `json:"edits,omitempty"`
}

type Publisher struct {
	mu       sync.Mutex
	messages []Message
}

func NewRedis() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Publish(msg Message) error {
	if msg.Version == "" {
		msg.Version = Version
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append(p.messages, msg)
	return nil
}

func (p *Publisher) Messages() []Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Message, len(p.messages))
	copy(out, p.messages)
	return out
}

type Stream struct {
	client *redis.Client
	stream string
}

func NewStream(queueURL, stream string) (*Stream, error) {
	options, err := redis.ParseURL(queueURL)
	if err != nil {
		return nil, fmt.Errorf("parse queue url: %w", err)
	}
	if strings.TrimSpace(stream) == "" {
		stream = "order-fill:jobs"
	}
	return &Stream{client: redis.NewClient(options), stream: stream}, nil
}

func (s *Stream) Publish(msg Message) error {
	if msg.Version == "" {
		msg.Version = Version
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: s.stream,
		Values: map[string]any{"payload": string(payload)},
	}).Err()
}
