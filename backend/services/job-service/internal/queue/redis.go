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

type OrderProfile struct {
	LegalName           string       `json:"legal_name,omitempty"`
	Consignee           string       `json:"consignee,omitempty"`
	Address             string       `json:"address,omitempty"`
	ContactName         string       `json:"contact_name,omitempty"`
	ContactPhone        string       `json:"contact_phone,omitempty"`
	Carrier             string       `json:"carrier,omitempty"`
	DeliveryPayer       string       `json:"delivery_payer,omitempty"`
	DeliveryDestination string       `json:"delivery_destination,omitempty"`
	BrandTerms          []BrandTerms `json:"brand_terms,omitempty"`
}

type BrandTerms struct {
	Brand               string `json:"brand"`
	DealerName          string `json:"dealer_name,omitempty"`
	PaymentMethod       string `json:"payment_method,omitempty"`
	PaymentControl      string `json:"payment_control,omitempty"`
	CustomerType        string `json:"customer_type,omitempty"`
	DiscountBasisPoints int32  `json:"discount_basis_points,omitempty"`
	DiscountSet         bool   `json:"discount_set,omitzero"`
}

type Message struct {
	Version      string       `json:"version"`
	JobID        string       `json:"job_id"`
	Type         string       `json:"type"`
	Stage        string       `json:"stage"`
	Brand        string       `json:"brand,omitempty"`
	MatchingMode string       `json:"matching_mode"`
	CompanyID    string       `json:"company_id,omitempty"`
	OrderProfile OrderProfile `json:"order_profile,omitzero"`
	Inputs       []Input      `json:"inputs"`
	Edits        []Edit       `json:"edits,omitempty"`
}

type Publisher struct {
	mu       sync.Mutex
	messages []Message
}

func NewRedis() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Publish(_ context.Context, msg Message) error {
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

func (s *Stream) Publish(ctx context.Context, msg Message) error {
	if msg.Version == "" {
		msg.Version = Version
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: s.stream,
		Values: map[string]any{"payload": string(payload)},
	}).Err()
}
