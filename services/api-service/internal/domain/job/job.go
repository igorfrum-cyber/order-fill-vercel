// Package job holds the api-service domain model. It depends on nothing but the
// standard library so the rules stay testable and free of transport or storage
// concerns.
package job

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
)

var (
	// ErrInvalid marks a rule violation caused by the caller.
	ErrInvalid = errors.New("invalid job")
	// ErrNotFound marks a job or report that does not exist.
	ErrNotFound = errors.New("job was not found")
)

type Type string

const (
	TypeOrderFill  Type = "order_fill"
	TypeNorthMerge Type = "north_merge"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusProcessing  Status = "processing"
	StatusNeedsReview Status = "needs_review"
	StatusFinalizing  Status = "finalizing"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

// Role describes what an uploaded file means to the processing pipeline.
type Role string

const (
	RoleSource Role = "source"
	RoleBlank  Role = "blank"
)

type Job struct {
	ID              string
	Type            Type
	Status          Status
	Brand           string
	OrderMonth      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Failure         *Failure
	InputFiles      []InputFile
	OutputFiles     []OutputFile
	Progress        float64
	ProgressMessage string
}

type Failure struct {
	Code    string
	Message string
}

type InputFile struct {
	ID          string
	Role        Role
	Name        string
	ContentType string
	SizeBytes   int64
	StorageKey  string
}

type OutputFile struct {
	ID          string
	Label       string
	Name        string
	ContentType string
	SizeBytes   int64
	StorageKey  string
}

// Upload is a file handed over by the transport layer before it is stored.
type Upload struct {
	Role        Role
	Name        string
	ContentType string
	Content     []byte
}

// NewJob builds a queued job and enforces the creation invariants.
func NewJob(id string, jobType Type, brand string, orderMonth string, now time.Time, files []InputFile) (Job, error) {
	if strings.TrimSpace(id) == "" {
		return Job{}, fmt.Errorf("%w: job id is required", ErrInvalid)
	}
	if jobType != TypeOrderFill && jobType != TypeNorthMerge {
		return Job{}, fmt.Errorf("%w: unsupported job type %q", ErrInvalid, jobType)
	}
	if strings.TrimSpace(brand) == "" {
		return Job{}, fmt.Errorf("%w: brand is required", ErrInvalid)
	}
	if jobType == TypeOrderFill && strings.TrimSpace(orderMonth) == "" {
		return Job{}, fmt.Errorf("%w: order_month is required", ErrInvalid)
	}
	if len(files) == 0 {
		return Job{}, fmt.Errorf("%w: at least one file is required", ErrInvalid)
	}
	timestamp := now.UTC()
	return Job{
		ID:          id,
		Type:        jobType,
		Status:      StatusQueued,
		Brand:       strings.TrimSpace(brand),
		OrderMonth:  strings.TrimSpace(orderMonth),
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
		InputFiles:  files,
		OutputFiles: []OutputFile{},
	}, nil
}

// ValidateUploads checks the files a caller sent before anything is persisted.
func ValidateUploads(jobType Type, uploads []Upload) error {
	if len(uploads) == 0 {
		return fmt.Errorf("%w: at least one file is required", ErrInvalid)
	}
	blanks := 0
	sources := 0
	for _, upload := range uploads {
		if strings.TrimSpace(upload.Name) == "" {
			return fmt.Errorf("%w: file name is required", ErrInvalid)
		}
		if len(upload.Content) == 0 {
			return fmt.Errorf("%w: file %q is empty", ErrInvalid, upload.Name)
		}
		if !hasWorkbookExtension(upload.Name) {
			return fmt.Errorf("%w: file %q must be .xlsx or .xlsm", ErrInvalid, upload.Name)
		}
		switch upload.Role {
		case RoleBlank:
			blanks++
		case RoleSource:
			sources++
		default:
			return fmt.Errorf("%w: unsupported file role %q", ErrInvalid, upload.Role)
		}
	}
	if blanks == 0 {
		return fmt.Errorf("%w: blank_files is required", ErrInvalid)
	}
	if jobType == TypeOrderFill && sources != 1 {
		return fmt.Errorf("%w: exactly one source_file is required", ErrInvalid)
	}
	return nil
}

// StorageKey is the object-storage location of an uploaded input file.
func StorageKey(jobID string, index int, fileName string) string {
	return fmt.Sprintf("jobs/%s/inputs/%d-%s", jobID, index, SafeFileName(fileName))
}

// OutputStorageKey is the object-storage location of a generated file.
func OutputStorageKey(jobID string, fileName string) string {
	return fmt.Sprintf("jobs/%s/outputs/%s", jobID, SafeFileName(fileName))
}

// ArchiveFileName is the download name for every generated workbook packed together.
func ArchiveFileName(brand string, orderMonth string) string {
	safeBrand := SafeFileName(brand)
	if safeBrand == "" || safeBrand == "file" {
		safeBrand = "order"
	}
	month := strings.TrimSpace(orderMonth)
	if month == "" {
		return safeBrand + ".zip"
	}
	return safeBrand + "_" + month + ".zip"
}

// SafeFileName strips any directory component a browser may have sent.
func SafeFileName(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	base := path.Base(normalized)
	if base == "." || base == "/" {
		return "file"
	}
	return base
}

func hasWorkbookExtension(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(lower, ".xlsx") || strings.HasSuffix(lower, ".xlsm")
}

// Fail marks the job as failed with a machine readable code.
func (j *Job) Fail(code string, message string, now time.Time) {
	j.Status = StatusFailed
	j.Failure = &Failure{Code: code, Message: message}
	j.UpdatedAt = now.UTC()
}

// CanAcceptEdits reports whether manual edits may still be submitted.
func (j Job) CanAcceptEdits() bool {
	return j.Status == StatusNeedsReview || j.Status == StatusCompleted
}
