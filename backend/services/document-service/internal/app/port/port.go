// Package port declares the outbound dependencies of the worker use cases.
package port

import (
	"context"
	"time"

	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

// Stage tells the worker which part of the pipeline to run.
type Stage string

const (
	StageProcess  Stage = "process"
	StageFinalize Stage = "finalize"
)

// Roles of the uploaded input files.
const (
	RoleSource     = "source"
	RoleBlank      = "blank"
	RoleBlankHome  = "blank-home"
	RoleBlankProff = "blank-proff"
	RoleWarehouse  = "warehouse"
)

// JobMessage is the queue contract published by job-service.
type JobMessage struct {
	JobID        string        `json:"job_id"`
	Type         string        `json:"type"`
	Stage        Stage         `json:"stage"`
	Brand        string        `json:"brand"`
	OrderMonth   string        `json:"order_month"`
	MatchingMode string        `json:"matching_mode,omitempty"`
	CompanyID    string        `json:"company_id,omitempty"`
	OrderProfile OrderProfile  `json:"order_profile,omitzero"`
	Inputs       []MessageFile `json:"inputs"`
	Edits        []MessageEdit `json:"edits,omitempty"`
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
// job-service owns the schema; the worker only advances a job it was given.
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
