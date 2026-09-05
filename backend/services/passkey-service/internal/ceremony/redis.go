package ceremony

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"order-fill/backend/services/passkey-service/internal/domain"
)

const keyPrefix = "passkey:ceremony:"

// Store keeps WebAuthn ceremony sessions. Empty redisURL keeps them in process
// for tests; compose points QUEUE_URL / REDIS_URL at Redis SET+EXPIRE.
type Store struct {
	mu     sync.Mutex
	items  map[string]domain.PasskeyChallenge
	client *redis.Client
}

func NewRedis() *Store {
	return &Store{items: map[string]domain.PasskeyChallenge{}}
}

func Open(redisURL string) (*Store, error) {
	if redisURL == "" {
		return NewRedis(), nil
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Store{client: redis.NewClient(opt)}, nil
}

func (s *Store) Put(ch domain.PasskeyChallenge) {
	if s.client != nil {
		raw, err := json.Marshal(ch)
		if err != nil {
			return
		}
		ttl := time.Until(ch.ExpiresAt)
		if ttl <= 0 {
			ttl = time.Minute
		}
		_ = s.client.Set(context.Background(), keyPrefix+ch.ID, raw, ttl).Err()
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[ch.ID] = ch
}

func (s *Store) Consume(id string, now time.Time) (domain.PasskeyChallenge, error) {
	if s.client != nil {
		key := keyPrefix + id
		raw, err := s.client.GetDel(context.Background(), key).Bytes()
		if err != nil {
			return domain.PasskeyChallenge{}, domain.ErrUnauthorized
		}
		var ch domain.PasskeyChallenge
		if json.Unmarshal(raw, &ch) != nil || !ch.ExpiresAt.After(now) {
			return domain.PasskeyChallenge{}, domain.ErrUnauthorized
		}
		return ch, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.items[id]
	if !ok || !ch.ExpiresAt.After(now) {
		return domain.PasskeyChallenge{}, domain.ErrUnauthorized
	}
	delete(s.items, id)
	return ch, nil
}
