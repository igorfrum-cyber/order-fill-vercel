package domain

import (
	"fmt"
	"path"
	"strings"
	"time"
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

type Role string

const (
	RoleSource    Role = "source"
	RoleBlank     Role = "blank"
	RoleWarehouse Role = "warehouse"
)

type Job struct {
	ID              string
	Type            Type
	Status          Status
	OwnerUserID     string
	CompanyID       string
	MatchingMode    MatchingMode
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ErrorMessage    string
	Progress        float64
	ProgressMessage string
	InputFiles      []FileRef
	OutputFiles     []FileRef
}

type FileRef struct {
	ID          string
	JobID       string
	Kind        string
	ObjectKey   string
	Name        string
	ContentType string
}

type UploadMeta struct {
	Role Role
	Name string
}

func ParseType(raw string) (Type, error) {
	switch Type(raw) {
	case TypeOrderFill, TypeNorthMerge:
		return Type(raw), nil
	default:
		return "", fmt.Errorf("%w: unsupported job type %q", ErrInvalid, raw)
	}
}

func NewJob(id string, jobType Type, ownerUserID, companyID string, mode MatchingMode, now time.Time, files []FileRef) (Job, error) {
	if strings.TrimSpace(id) == "" {
		return Job{}, fmt.Errorf("%w: job id is required", ErrInvalid)
	}
	if jobType != TypeOrderFill && jobType != TypeNorthMerge {
		return Job{}, fmt.Errorf("%w: unsupported job type %q", ErrInvalid, jobType)
	}
	if strings.TrimSpace(ownerUserID) == "" || strings.TrimSpace(companyID) == "" {
		return Job{}, fmt.Errorf("%w: job owner is required", ErrInvalid)
	}
	if len(files) == 0 {
		return Job{}, fmt.Errorf("%w: at least one file is required", ErrInvalid)
	}
	if mode == "" {
		mode = MatchingModeStandard
	}
	timestamp := now.UTC()
	return Job{
		ID:           id,
		Type:         jobType,
		Status:       StatusQueued,
		OwnerUserID:  ownerUserID,
		CompanyID:    companyID,
		MatchingMode: mode,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
		InputFiles:   files,
		OutputFiles:  []FileRef{},
	}, nil
}

func ValidateUploads(jobType Type, uploads []UploadMeta) error {
	if len(uploads) == 0 {
		return fmt.Errorf("%w: at least one file is required", ErrInvalid)
	}
	blanks := 0
	sources := 0
	warehouses := 0
	seen := map[string]struct{}{}
	for _, upload := range uploads {
		if strings.TrimSpace(upload.Name) == "" {
			return fmt.Errorf("%w: file name is required", ErrInvalid)
		}
		if !hasWorkbookExtension(upload.Name) {
			return fmt.Errorf("%w: file %q must be .xlsx or .xlsm", ErrInvalid, upload.Name)
		}
		key := strings.ToLower(strings.TrimSpace(upload.Name))
		if _, dup := seen[key]; dup {
			return fmt.Errorf("%w: файл %q загружен дважды", ErrInvalid, upload.Name)
		}
		seen[key] = struct{}{}
		switch upload.Role {
		case RoleBlank:
			blanks++
		case RoleSource:
			sources++
		case RoleWarehouse:
			warehouses++
		default:
			return fmt.Errorf("%w: unsupported file role %q", ErrInvalid, upload.Role)
		}
	}
	if warehouses > 1 {
		return fmt.Errorf("%w: at most one warehouse file is allowed", ErrInvalid)
	}
	switch jobType {
	case TypeOrderFill:
		if blanks == 0 {
			return fmt.Errorf("%w: blank_files is required", ErrInvalid)
		}
		if blanks > 2 {
			return fmt.Errorf("%w: order fill accepts at most two blank_files", ErrInvalid)
		}
		if sources != 1 {
			return fmt.Errorf("%w: exactly one source_file is required", ErrInvalid)
		}
	case TypeNorthMerge:
		if blanks == 0 {
			return fmt.Errorf("%w: blank_files is required", ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported job type %q", ErrInvalid, jobType)
	}
	return nil
}

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

func (j Job) CanAcceptEdits() bool {
	return j.Status == StatusNeedsReview || j.Status == StatusCompleted
}

func (j *Job) Fail(message string, now time.Time) {
	j.Status = StatusFailed
	j.ErrorMessage = message
	j.UpdatedAt = now.UTC()
}
