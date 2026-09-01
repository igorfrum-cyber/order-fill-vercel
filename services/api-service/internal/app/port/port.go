// Package port declares the outbound dependencies the use cases need.
// Interfaces live with their consumer, so adapters depend on the application
// layer and never the other way around.
package port

import (
	"context"
	"time"

	"order-fill/services/api-service/internal/domain/job"
)

// JobRepository persists job metadata.
type JobRepository interface {
	Create(ctx context.Context, entity job.Job) error
	Get(ctx context.Context, id string) (job.Job, error)
	UpdateStatus(ctx context.Context, id string, status job.Status, updatedAt time.Time) (job.Job, error)
	List(ctx context.Context, filter JobListFilter) ([]JobListRow, error)
}

// ReportReader reads the review report produced by document-service.
type ReportReader interface {
	Report(ctx context.Context, jobID string) (job.Report, error)
}

// ObjectStore stores uploaded and generated files.
type ObjectStore interface {
	Put(ctx context.Context, key string, contentType string, content []byte) error
	Get(ctx context.Context, key string) (Object, error)
}

// Object is a stored file returned by the object store.
type Object struct {
	Content     []byte
	ContentType string
}

// JobPublisher hands work over to document-service.
type JobPublisher interface {
	Publish(ctx context.Context, message JobMessage) error
}

// Stage tells the worker which part of the pipeline to run.
type Stage string

const (
	StageProcess  Stage = "process"
	StageFinalize Stage = "finalize"
)

// JobMessage is the queue contract between api-service and document-service.
type JobMessage struct {
	JobID      string        `json:"job_id"`
	Type       string        `json:"type"`
	Stage      Stage         `json:"stage"`
	Brand      string        `json:"brand"`
	OrderMonth string        `json:"order_month"`
	Inputs     []MessageFile `json:"inputs"`
	Edits      []MessageEdit `json:"edits,omitempty"`
}

// MessageFile points the worker at an input file in object storage.
type MessageFile struct {
	Role       string `json:"role"`
	Name       string `json:"name"`
	StorageKey string `json:"storage_key"`
}

// MessageEdit is a reviewer correction forwarded to the worker.
type MessageEdit struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Comment string `json:"comment,omitempty"`
}

// Clock returns the current time. Injected so use cases stay deterministic.
type Clock func() time.Time

// IDGenerator returns a new unique job identifier.
type IDGenerator func() string

// Metrics records counters for observability.
type Metrics interface {
	AddJobCreated()
	AddJobFailed()
	AddBytesStored(value int64)
}
