package grpcjobs

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"order-fill/backend/pkg/grpcutil"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/document-service/internal/app/port"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

type adapterFilesClient struct {
	filesv1.FileServiceClient
	objects      map[string][]byte
	getErr       error
	putErr       error
	getRequests  []*filesv1.GetObjectRequest
	putRequests  []*filesv1.PutObjectRequest
	workerTokens []string
}

func (f *adapterFilesClient) GetObject(ctx context.Context, req *filesv1.GetObjectRequest, _ ...grpc.CallOption) (*filesv1.GetObjectResponse, error) {
	f.getRequests = append(f.getRequests, req)
	f.workerTokens = append(f.workerTokens, outgoingWorkerToken(ctx))
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &filesv1.GetObjectResponse{
		Object: &filesv1.ObjectMeta{Key: req.GetKey(), ContentType: "application/json"},
		Body:   f.objects[req.GetKey()],
	}, nil
}

func (f *adapterFilesClient) PutObject(ctx context.Context, req *filesv1.PutObjectRequest, _ ...grpc.CallOption) (*filesv1.PutObjectResponse, error) {
	f.putRequests = append(f.putRequests, req)
	f.workerTokens = append(f.workerTokens, outgoingWorkerToken(ctx))
	if f.putErr != nil {
		return nil, f.putErr
	}
	if f.objects == nil {
		f.objects = make(map[string][]byte)
	}
	f.objects[req.GetKey()] = append([]byte(nil), req.GetBody()...)
	return &filesv1.PutObjectResponse{}, nil
}

type adapterJobsClient struct {
	jobsv1.JobServiceClient
	progressRequests []*jobsv1.UpdateProgressRequest
	failRequest      *jobsv1.FailJobRequest
	completeRequest  *jobsv1.CompleteJobRequest
	progressErr      error
	failErr          error
	completeErr      error
}

func (j *adapterJobsClient) UpdateProgress(_ context.Context, req *jobsv1.UpdateProgressRequest, _ ...grpc.CallOption) (*jobsv1.UpdateProgressResponse, error) {
	j.progressRequests = append(j.progressRequests, req)
	return &jobsv1.UpdateProgressResponse{}, j.progressErr
}

func (j *adapterJobsClient) FailJob(_ context.Context, req *jobsv1.FailJobRequest, _ ...grpc.CallOption) (*jobsv1.FailJobResponse, error) {
	j.failRequest = req
	return &jobsv1.FailJobResponse{}, j.failErr
}

func (j *adapterJobsClient) CompleteJob(_ context.Context, req *jobsv1.CompleteJobRequest, _ ...grpc.CallOption) (*jobsv1.CompleteJobResponse, error) {
	j.completeRequest = req
	return &jobsv1.CompleteJobResponse{}, j.completeErr
}

func TestFilesGetAndPutCarryWorkerAndCompanyMetadata(t *testing.T) {
	t.Parallel()

	filesClient := &adapterFilesClient{objects: map[string][]byte{"reports/report.json": []byte(`{"ok":true}`)}}
	files := Files{API: filesClient, Token: "document-worker-secret"}
	ctx := grpcutil.WithCompany(t.Context(), "company-1")
	body, err := files.Get(ctx, "reports/report.json")
	if err != nil {
		t.Fatal(err)
	}
	body[0] = 'X'
	if filesClient.objects["reports/report.json"][0] != '{' {
		t.Fatal("Get returned the storage buffer instead of a copy")
	}
	if got := filesClient.getRequests[0].GetMeta().GetCompanyId(); got != "company-1" {
		t.Fatalf("company metadata = %q", got)
	}
	if got := filesClient.workerTokens[0]; got != "document-worker-secret" {
		t.Fatalf("worker token = %q", got)
	}
	if err := files.Put(ctx, "jobs/job-1/report.json", "application/json", []byte(`{"rows":[]}`)); err != nil {
		t.Fatal(err)
	}
	put := filesClient.putRequests[0]
	if put.GetName() != "report.json" || put.GetContentType() != "application/json" || put.GetMeta().GetCompanyId() != "company-1" {
		t.Fatalf("put request = %+v", put)
	}
	if got := filesClient.workerTokens[1]; got != "document-worker-secret" {
		t.Fatalf("worker token = %q", got)
	}
}

func TestFilesGetAndPutPropagateErrors(t *testing.T) {
	t.Parallel()

	want := errors.New("file service unavailable")
	files := Files{API: &adapterFilesClient{getErr: want, putErr: want}}
	if _, err := files.Get(t.Context(), "key"); !errors.Is(err, want) {
		t.Fatalf("Get() error = %v", err)
	}
	if err := files.Put(t.Context(), "key", "text/plain", nil); !errors.Is(err, want) {
		t.Fatalf("Put() error = %v", err)
	}
}

func TestJobsUpdateProgressProcessingAndFailure(t *testing.T) {
	t.Parallel()

	api := &adapterJobsClient{}
	jobs := Jobs{API: api, Files: Files{Token: "worker-token"}}
	ctx := t.Context()
	if err := jobs.MarkProcessing(ctx, "job-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := jobs.MarkFailed(ctx, "job-1", "invalid", "broken workbook", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := jobs.SetProgress(ctx, "job-1", 0.75, "halfway", time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(api.progressRequests) != 2 || api.progressRequests[0].GetStatus() != "processing" ||
		api.progressRequests[1].GetProgress() != 0.75 || api.progressRequests[1].GetMessage() != "halfway" {
		t.Fatalf("progress requests = %+v", api.progressRequests)
	}
	if api.failRequest.GetJobId() != "job-1" || api.failRequest.GetErrorMessage() != "broken workbook" {
		t.Fatalf("failed job request = %+v", api.failRequest)
	}
}

func TestJobsUpdateProgressAndFailureReturnRPCError(t *testing.T) {
	t.Parallel()

	want := errors.New("jobs unavailable")
	jobs := Jobs{API: &adapterJobsClient{progressErr: want, failErr: want}}
	if err := jobs.MarkProcessing(t.Context(), "job-1", time.Time{}); !errors.Is(err, want) {
		t.Fatalf("MarkProcessing() error = %v", err)
	}
	if err := jobs.SetProgress(t.Context(), "job-1", 0.5, "working", time.Time{}); !errors.Is(err, want) {
		t.Fatalf("SetProgress() error = %v", err)
	}
	if err := jobs.MarkFailed(t.Context(), "job-1", "internal", "failed", time.Time{}); !errors.Is(err, want) {
		t.Fatalf("MarkFailed() error = %v", err)
	}
}

func TestJobsSaveResultPersistsOutputsAndReport(t *testing.T) {
	t.Parallel()

	key := "jobs/job-1/report.json"
	filesClient := &adapterFilesClient{objects: map[string][]byte{key: []byte(`{"rows":[{"key":"row-1","status":"matched","category":"to_order","blank_article":"A1","blank_name":"Cream"}],"summary":{}}`)}}
	api := &adapterJobsClient{}
	jobs := Jobs{API: api, Files: Files{API: filesClient, Token: "worker-token"}}
	outputs := []port.OutputFile{{ID: "out-1", Label: "filled", Name: "filled.xlsx", ContentType: "application/xlsx", StorageKey: "jobs/job-1/outputs/filled.xlsx"}}
	if err := jobs.SaveResult(t.Context(), "job-1", "completed", outputs, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if len(api.progressRequests) != 1 || api.progressRequests[0].GetStatus() != "finalizing" {
		t.Fatalf("finalizing progress = %+v", api.progressRequests)
	}
	var stored []port.OutputFile
	if err := json.Unmarshal(filesClient.objects["jobs/job-1/outputs.json"], &stored); err != nil || !reflect.DeepEqual(stored, outputs) {
		t.Fatalf("stored outputs = %+v, %v", stored, err)
	}
	request := api.completeRequest
	if request.GetJobId() != "job-1" || len(request.GetFiles()) != 1 || request.GetFiles()[0].GetId() != "out-1" || request.GetSummary().GetToOrder() != 1 || len(request.GetRows()) != 1 {
		t.Fatalf("complete job request = %+v", request)
	}
}

func TestJobsSaveResultWithoutReportAndReturnErrors(t *testing.T) {
	t.Parallel()

	filesClient := &adapterFilesClient{getErr: errors.New("report missing")}
	api := &adapterJobsClient{}
	jobs := Jobs{API: api, Files: Files{API: filesClient}}
	if err := jobs.SaveResult(t.Context(), "job-2", "failed", nil, time.Time{}); err != nil {
		t.Fatalf("SaveResult without report = %v", err)
	}
	if api.completeRequest.GetJobId() != "job-2" || api.completeRequest.GetSummary() != nil {
		t.Fatalf("completion request = %+v", api.completeRequest)
	}

	filesClient = &adapterFilesClient{putErr: errors.New("storage unavailable")}
	if err := (Jobs{API: api, Files: Files{API: filesClient}}).SaveResult(t.Context(), "job-3", "completed", nil, time.Time{}); err == nil {
		t.Fatal("SaveResult swallowed output write error")
	}

	api = &adapterJobsClient{completeErr: errors.New("completion unavailable")}
	if err := (Jobs{API: api, Files: Files{API: &adapterFilesClient{}}}).SaveResult(t.Context(), "job-4", "failed", nil, time.Time{}); err == nil {
		t.Fatal("SaveResult swallowed completion error")
	}
}

func TestJobsOutputsReadsAndDecodesStoredData(t *testing.T) {
	t.Parallel()

	want := []port.OutputFile{{ID: "out-1", Name: "filled.xlsx"}}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	filesClient := &adapterFilesClient{objects: map[string][]byte{"jobs/job-1/outputs.json": raw}}
	got, err := (Jobs{Files: Files{API: filesClient}}).Outputs(t.Context(), "job-1")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Outputs() = %+v, %v", got, err)
	}
	if _, err := (Jobs{Files: Files{API: &adapterFilesClient{getErr: errors.New("missing")}}}).Outputs(t.Context(), "job-1"); err == nil {
		t.Fatal("Outputs swallowed storage error")
	}
	if _, err := (Jobs{Files: Files{API: &adapterFilesClient{objects: map[string][]byte{"jobs/job-1/outputs.json": []byte("{")}}}}).Outputs(t.Context(), "job-1"); err == nil {
		t.Fatal("Outputs swallowed malformed JSON")
	}
}

func TestJobsSetIdentityAndReportsSaveLoad(t *testing.T) {
	t.Parallel()

	filesClient := &adapterFilesClient{}
	jobs := Jobs{Files: Files{API: filesClient}}
	if err := jobs.SetIdentity(t.Context(), "job-1", "klapp", "2026-10", time.Time{}); err != nil {
		t.Fatal(err)
	}
	var identity map[string]string
	if err := json.Unmarshal(filesClient.objects["jobs/job-1/identity.json"], &identity); err != nil || identity["brand"] != "klapp" || identity["order_month"] != "2026-10" {
		t.Fatalf("identity = %v, %v", identity, err)
	}

	reports := Reports{Files: Files{API: filesClient}}
	summary := orderfill.Summary{Brand: "klapp", Filled: 1}
	rows := []orderfill.ReportRow{{Key: "row-1", BlankArticle: "A1"}}
	if err := reports.Save(t.Context(), "job-1", summary, rows, time.Time{}); err != nil {
		t.Fatal(err)
	}
	gotSummary, gotRows, err := reports.Load(t.Context(), "job-1")
	if err != nil || !reflect.DeepEqual(gotSummary, summary) || !reflect.DeepEqual(gotRows, rows) {
		t.Fatalf("Load() = %+v, %+v, %v", gotSummary, gotRows, err)
	}
	if _, _, err := (Reports{Files: Files{API: &adapterFilesClient{getErr: errors.New("missing")}}}).Load(t.Context(), "job-1"); err == nil {
		t.Fatal("Load swallowed storage error")
	}
	if _, _, err := (Reports{Files: Files{API: &adapterFilesClient{objects: map[string][]byte{"jobs/job-1/report.json": []byte("{")}}}}).Load(t.Context(), "job-1"); err == nil {
		t.Fatal("Load swallowed malformed JSON")
	}
}

func outgoingWorkerToken(ctx context.Context) string {
	md, _ := metadata.FromOutgoingContext(ctx)
	values := md.Get(grpcutil.WorkerTokenMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
