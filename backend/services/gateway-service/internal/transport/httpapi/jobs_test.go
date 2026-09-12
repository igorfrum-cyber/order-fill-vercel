package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"

	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/gateway-service/internal/clients"
)

type uploadFileClient struct {
	filesv1.FileServiceClient
	requests []*filesv1.PutObjectRequest
}

func (c *uploadFileClient) PutObject(_ context.Context, req *filesv1.PutObjectRequest, _ ...grpc.CallOption) (*filesv1.PutObjectResponse, error) {
	c.requests = append(c.requests, req)
	id := "file-" + req.GetName()
	return &filesv1.PutObjectResponse{Object: &filesv1.ObjectMeta{Id: id}}, nil
}

func (c *uploadFileClient) GetObject(context.Context, *filesv1.GetObjectRequest, ...grpc.CallOption) (*filesv1.GetObjectResponse, error) {
	return nil, errors.New("not found")
}

type uploadJobClient struct {
	jobsv1.JobServiceClient
	request *jobsv1.CreateJobRequest
}

func (c *uploadJobClient) CreateJob(_ context.Context, req *jobsv1.CreateJobRequest, _ ...grpc.CallOption) (*jobsv1.CreateJobResponse, error) {
	c.request = req
	return &jobsv1.CreateJobResponse{Job: &jobsv1.Job{
		Id: "job-1", Type: req.GetType(), Status: "queued", OwnerUserId: "user-1", CompanyId: "company-1",
	}}, nil
}

func (c *uploadJobClient) ListFiles(context.Context, *jobsv1.ListFilesRequest, ...grpc.CallOption) (*jobsv1.ListFilesResponse, error) {
	return &jobsv1.ListFilesResponse{}, nil
}

func TestCreateOrderFillUploadsWarehouseWithItsOwnRole(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	addFile := func(field, name string) {
		t.Helper()
		part, err := form.CreateFormFile(field, name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, name); err != nil {
			t.Fatal(err)
		}
	}
	addFile("source_file", "office.xlsx")
	addFile("warehouse_file", "warehouse.xlsx")
	addFile("blank_files", "blank.xlsx")
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	files := &uploadFileClient{}
	jobs := &uploadJobClient{}
	api := &API{Clients: clients.Clients{Files: files, Jobs: jobs}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/order-fill", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req = req.WithContext(withUser(t.Context(), User{ID: "user-1", CompanyID: "company-1", Role: "purchaser"}))
	rec := httptest.NewRecorder()

	api.createOrderFill(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(files.requests) != 3 {
		t.Fatalf("uploads=%d want 3", len(files.requests))
	}
	wantPrefixes := []string{"source/", "warehouse/", "blank/"}
	for i, want := range wantPrefixes {
		if key := files.requests[i].GetKey(); !strings.HasPrefix(key, want) {
			t.Fatalf("upload %d key=%q want prefix %q", i, key, want)
		}
	}
	if jobs.request == nil {
		t.Fatal("job was not created")
	}
	wantIDs := []string{"file-office.xlsx", "file-warehouse.xlsx", "file-blank.xlsx"}
	if got := jobs.request.GetInputFileIds(); strings.Join(got, ",") != strings.Join(wantIDs, ",") {
		t.Fatalf("input ids=%v want %v", got, wantIDs)
	}
}
