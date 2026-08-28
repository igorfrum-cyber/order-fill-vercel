package jobs

import (
	"context"
	"sync"
)

type MemoryQueue struct {
	mu       sync.Mutex
	messages []JobMessage
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{}
}

func (q *MemoryQueue) Enqueue(_ context.Context, message JobMessage) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = append(q.messages, message)
	return nil
}

func (q *MemoryQueue) Messages() []JobMessage {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]JobMessage(nil), q.messages...)
}
