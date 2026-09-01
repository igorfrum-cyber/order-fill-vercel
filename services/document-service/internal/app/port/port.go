// Package port declares the outbound dependencies of the worker use cases.
package port

import (
	"context"
	"time"

	"order-fill/services/document-service/internal/domain/orderfill"
)

// Stage tells the worker which part of the pipeline to run.
type Stage string

const (
	StageProcess  Stage = "process"
	StageFinalize Stage = "finalize"
)

// Roles of the uploaded input files.
const (
	RoleSource = "source"
	RoleBlank  = "blank"
)

// JobMessage is the queue contract published by api-service.
type JobMessage struct {
	JobID      string        `json:"job_id"`
	Type       string        `json:"type"`
	Stage      Stage         `json:"stage"`
	Brand      string        `json:"brand"`
	OrderMonth string        `json:"order_month"`
	Inputs     []MessageFile `json:"inputs"`
	Edits      []MessageEdit `json:"edits,omitempty"`
}

// MessageFile points at an input file in object storage.
type MessageFile struct {
	Role       string `json:"role"`
	Name       string `json:"name"`
	StorageKey string `json:"storage_key"`
}

// MessageEdit is a reviewer correction.
type MessageEdit struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Comment string `json:"comment,omitempty"`
}

// ObjectStore reads uploaded files and stores generated ones.
type ObjectStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, contentType string, content []byte) error
}

// OutputFile is a generated document recorded on the job.
type OutputFile struct {
	ID          string
	Label       string
	Name        string
	ContentType string
	SizeBytes   int64
	StorageKey  string
}

// JobStore is the narrow slice of the shared job table the worker touches.
// api-service owns the schema; the worker only advances a job it was given.
type JobStore interface {
	MarkProcessing(ctx context.Context, jobID string, at time.Time) error
	MarkFailed(ctx context.Context, jobID string, code string, message string, at time.Time) error
	SaveResult(ctx context.Context, jobID string, status string, outputs []OutputFile, at time.Time) error
	SetIdentity(ctx context.Context, jobID string, brand string, orderMonth string, at time.Time) error
	SetProgress(ctx context.Context, jobID string, fraction float64, message string, at time.Time) error
	Outputs(ctx context.Context, jobID string) ([]OutputFile, error)
}

// ReportStore persists and reloads the reviewable report.
type ReportStore interface {
	Save(ctx context.Context, jobID string, summary orderfill.Summary, rows []orderfill.ReportRow, at time.Time) error
	Load(ctx context.Context, jobID string) (orderfill.Summary, []orderfill.ReportRow, error)
}

// Metrics records worker counters.
type Metrics interface {
	AddJobCompleted(durationMS int64)
	AddJobFailed(durationMS int64)
}

// Clock returns the current time.
type Clock func() time.Time
