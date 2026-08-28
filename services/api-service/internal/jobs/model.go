package jobs

import "time"

type JobType string

const (
	JobTypeOrderFill  JobType = "order_fill"
	JobTypeNorthMerge JobType = "north_merge"
)

type JobStatus string

const (
	JobStatusQueued      JobStatus = "queued"
	JobStatusProcessing  JobStatus = "processing"
	JobStatusNeedsReview JobStatus = "needs_review"
	JobStatusFinalizing  JobStatus = "finalizing"
	JobStatusCompleted   JobStatus = "completed"
	JobStatusFailed      JobStatus = "failed"
)

type Job struct {
	ID          string       `json:"id"`
	Type        JobType      `json:"type"`
	Status      JobStatus    `json:"status"`
	Brand       string       `json:"brand,omitempty"`
	OrderMonth  string       `json:"order_month,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Error       *JobError    `json:"error,omitempty"`
	InputFiles  []StoredFile `json:"input_files"`
	OutputFiles []OutputFile `json:"output_files"`
}

type JobError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type StoredFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Key         string `json:"-"`
}

type OutputFile struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	DownloadURL string `json:"download_url"`
}

type JobMessage struct {
	JobID string  `json:"job_id"`
	Type  JobType `json:"type"`
}
