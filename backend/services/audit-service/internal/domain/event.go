package domain

import "time"

type Event struct {
	ID        string
	Type      string
	ActorID   string
	CompanyID string
	JobID     string
	CreatedAt time.Time
	Payload   string
}
