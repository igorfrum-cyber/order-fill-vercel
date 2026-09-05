package grpcjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"time"

	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/document-service/internal/app/port"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

type Files struct {
	API filesv1.FileServiceClient
}

func (s Files) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.API.GetObject(ctx, &filesv1.GetObjectRequest{Key: key})
	if err != nil {
		return nil, err
	}
	return bytes.Clone(resp.GetBody()), nil
}

func (s Files) Put(ctx context.Context, key string, contentType string, content []byte) error {
	_, err := s.API.PutObject(ctx, &filesv1.PutObjectRequest{
		Key: key, Name: path.Base(key), ContentType: contentType, Body: content,
	})
	return err
}

type Jobs struct {
	API   jobsv1.JobServiceClient
	Files Files
}

func (j Jobs) MarkProcessing(ctx context.Context, jobID string, _ time.Time) error {
	_, err := j.API.UpdateProgress(ctx, &jobsv1.UpdateProgressRequest{JobId: jobID, Status: "processing", Message: "processing"})
	return err
}

func (j Jobs) MarkFailed(ctx context.Context, jobID string, _ string, message string, _ time.Time) error {
	_, err := j.API.FailJob(ctx, &jobsv1.FailJobRequest{JobId: jobID, ErrorMessage: message})
	return err
}

func (j Jobs) SaveResult(ctx context.Context, jobID string, status string, outputs []port.OutputFile, _ time.Time) error {
	listed, err := json.Marshal(outputs)
	if err != nil {
		return err
	}
	if err := j.Files.Put(ctx, "jobs/"+jobID+"/outputs.json", "application/json", listed); err != nil {
		return err
	}
	files := make([]*jobsv1.FileRef, 0, len(outputs))
	for _, out := range outputs {
		files = append(files, &jobsv1.FileRef{
			Id: out.ID, JobId: jobID, Kind: out.Label, ObjectKey: out.StorageKey,
			Name: out.Name, ContentType: out.ContentType,
		})
	}
	if status == "completed" {
		_, _ = j.API.UpdateProgress(ctx, &jobsv1.UpdateProgressRequest{JobId: jobID, Status: "finalizing"})
	}
	req := &jobsv1.CompleteJobRequest{JobId: jobID, Files: files}
	if raw, err := j.Files.Get(ctx, "jobs/"+jobID+"/report.json"); err == nil {
		req.Summary, req.Rows = completeReport(raw)
	}
	_, err = j.API.CompleteJob(ctx, req)
	return err
}

func (j Jobs) Outputs(ctx context.Context, jobID string) ([]port.OutputFile, error) {
	raw, err := j.Files.Get(ctx, "jobs/"+jobID+"/outputs.json")
	if err != nil {
		return nil, err
	}
	var outputs []port.OutputFile
	if err := json.Unmarshal(raw, &outputs); err != nil {
		return nil, err
	}
	return outputs, nil
}

func (j Jobs) SetIdentity(ctx context.Context, jobID string, brand string, orderMonth string, _ time.Time) error {
	payload, err := json.Marshal(map[string]string{"brand": brand, "order_month": orderMonth})
	if err != nil {
		return err
	}
	return j.Files.Put(ctx, "jobs/"+jobID+"/identity.json", "application/json", payload)
}

func (j Jobs) SetProgress(ctx context.Context, jobID string, fraction float64, message string, _ time.Time) error {
	_, err := j.API.UpdateProgress(ctx, &jobsv1.UpdateProgressRequest{
		JobId: jobID, Status: "processing", Message: message, Progress: fraction,
	})
	return err
}

type Reports struct {
	Files Files
}

func (r Reports) Save(ctx context.Context, jobID string, summary orderfill.Summary, rows []orderfill.ReportRow, _ time.Time) error {
	payload, err := json.Marshal(map[string]any{"job_id": jobID, "summary": summary, "rows": rows})
	if err != nil {
		return err
	}
	return r.Files.Put(ctx, "jobs/"+jobID+"/report.json", "application/json", payload)
}

func (r Reports) Load(ctx context.Context, jobID string) (orderfill.Summary, []orderfill.ReportRow, error) {
	raw, err := r.Files.Get(ctx, "jobs/"+jobID+"/report.json")
	if err != nil {
		return orderfill.Summary{}, nil, err
	}
	var payload struct {
		Summary orderfill.Summary     `json:"summary"`
		Rows    []orderfill.ReportRow `json:"rows"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return orderfill.Summary{}, nil, err
	}
	return payload.Summary, payload.Rows, nil
}
