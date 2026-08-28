package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"order-fill/services/api-service/internal/jobs"
)

func TestCreateOrderFillJobAcceptsMultipartUpload(t *testing.T) {
	creator := &recordingJobCreator{
		job: jobs.Job{
			ID:         "job-1",
			Type:       jobs.JobTypeOrderFill,
			Status:     jobs.JobStatusQueued,
			Brand:      "angiopharm",
			OrderMonth: "2026-08",
			CreatedAt:  time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC),
		},
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("brand", "angiopharm")
	_ = writer.WriteField("order_month", "2026-08")
	source, err := writer.CreateFormFile("source_file", "source.xlsx")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	_, _ = source.Write([]byte("source"))
	blank, err := writer.CreateFormFile("blank_files", "blank.xlsx")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	_, _ = blank.Write([]byte("blank"))
	_ = writer.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/order-fill", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	NewRouter(WithJobCreator(creator)).ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusAccepted, response.Code, response.Body.String())
	}
	if creator.command.Type != jobs.JobTypeOrderFill {
		t.Fatalf("expected order-fill command, got %q", creator.command.Type)
	}
	if len(creator.command.Files) != 2 {
		t.Fatalf("expected source and blank files, got %d", len(creator.command.Files))
	}
	var payload jobs.Job
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if payload.ID != "job-1" || payload.Status != jobs.JobStatusQueued {
		t.Fatalf("unexpected response payload %#v", payload)
	}
}

type recordingJobCreator struct {
	command jobs.CreateJobCommand
	job     jobs.Job
}

func (c *recordingJobCreator) CreateJob(_ context.Context, command jobs.CreateJobCommand) (jobs.Job, error) {
	c.command = command
	return c.job, nil
}
