package grpcapi

import (
	"context"
	"math"
	"time"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/job-service/internal/domain"
	"order-fill/backend/services/job-service/internal/service/jobs"
)

type Server struct {
	jobsv1.UnimplementedJobServiceServer
	svc         *jobs.Service
	actors      ActorLookup
	workerToken string
}

type ActorLookup interface {
	Actor(ctx context.Context, userID string) (domain.Actor, error)
}

func NewServer(svc *jobs.Service, actors ActorLookup, workerToken string) *Server {
	return &Server{svc: svc, actors: actors, workerToken: workerToken}
}

func (s *Server) requireWorker(ctx context.Context) error {
	if s.workerToken == "" {
		return nil
	}
	if !grpcutil.WorkerAuthorized(ctx, s.workerToken) {
		return domain.ErrUnauthorized
	}
	return nil
}

func New(handler jobsv1.JobServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		jobsv1.RegisterJobServiceServer(s, handler)
	}
	return s
}

func (s *Server) actorFrom(ctx context.Context, meta *commonv1.RequestMeta) (domain.Actor, error) {
	userID := ""
	companyID := ""
	if meta != nil {
		userID = meta.GetActorUserId()
		companyID = meta.GetCompanyId()
	}
	if s.actors != nil {
		return s.actors.Actor(ctx, userID)
	}
	if userID == "" {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	// ponytail: without identity, ignore x-actor-role and stay purchaser. Look up role when IDENTITY_GRPC_ADDR is set.
	return domain.Actor{UserID: userID, CompanyID: companyID, Role: domain.RolePurchaser}, nil
}

func protoJob(job domain.Job) *jobsv1.Job {
	mode := commonv1.MatchingMode_MATCHING_MODE_STANDARD
	if job.MatchingMode == domain.MatchingModeSmart {
		mode = commonv1.MatchingMode_MATCHING_MODE_SMART
	}
	return &jobsv1.Job{
		Id:              job.ID,
		Type:            string(job.Type),
		Status:          string(job.Status),
		OwnerUserId:     job.OwnerUserID,
		CompanyId:       job.CompanyID,
		MatchingMode:    mode,
		CreatedAt:       job.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       job.UpdatedAt.UTC().Format(time.RFC3339),
		ErrorMessage:    job.ErrorMessage,
		Progress:        job.Progress,
		ProgressMessage: job.ProgressMessage,
	}
}

func (s *Server) CreateJob(ctx context.Context, req *jobsv1.CreateJobRequest) (*jobsv1.CreateJobResponse, error) {
	actor, err := s.actorFrom(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	jobType, err := domain.ParseType(req.GetType())
	if err != nil {
		return nil, err
	}
	job, err := s.svc.Create(ctx, actor, jobType, req.GetInputFileIds(), req.GetBrand())
	if err != nil {
		return nil, err
	}
	return &jobsv1.CreateJobResponse{Job: protoJob(job)}, nil
}

func (s *Server) GetJob(ctx context.Context, req *jobsv1.GetJobRequest) (*jobsv1.GetJobResponse, error) {
	actor, err := s.actorFrom(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	job, err := s.svc.Get(ctx, actor, req.GetJobId())
	if err != nil {
		return nil, err
	}
	return &jobsv1.GetJobResponse{Job: protoJob(job)}, nil
}

func (s *Server) ListJobs(ctx context.Context, req *jobsv1.ListJobsRequest) (*jobsv1.ListJobsResponse, error) {
	actor, err := s.actorFrom(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	items, err := s.svc.List(ctx, actor)
	if err != nil {
		return nil, err
	}
	out := make([]*jobsv1.Job, 0, len(items))
	for _, item := range items {
		out = append(out, protoJob(item))
	}
	return &jobsv1.ListJobsResponse{Jobs: out}, nil
}

func (s *Server) CompleteJob(ctx context.Context, req *jobsv1.CompleteJobRequest) (*jobsv1.CompleteJobResponse, error) {
	if err := s.requireWorker(ctx); err != nil {
		return nil, err
	}
	files := make([]domain.FileRef, 0, len(req.GetFiles()))
	for _, file := range req.GetFiles() {
		files = append(files, domain.FileRef{
			ID: file.GetId(), JobID: file.GetJobId(), Kind: file.GetKind(),
			ObjectKey: file.GetObjectKey(), Name: file.GetName(), ContentType: file.GetContentType(),
		})
	}
	report := domain.Report{Summary: domain.ReportSummary{
		NeedsDecision:     int(req.GetSummary().GetNeedsDecision()),
		NotInSource:       int(req.GetSummary().GetNotInSource()),
		CheckNameOrVolume: int(req.GetSummary().GetCheckNameOrVolume()),
		NotInBlank:        int(req.GetSummary().GetNotInBlank()),
		ToOrder:           int(req.GetSummary().GetToOrder()),
		OrderNotNeeded:    int(req.GetSummary().GetOrderNotNeeded()),
	}}
	for _, row := range req.GetRows() {
		report.Rows = append(report.Rows, domain.ReportRow{
			ID: row.GetId(), Category: domain.ReportCategory(reportCategoryName(row.GetCategory())),
			Article: row.GetArticle(), Name: row.GetName(), Reasons: append([]string{}, row.GetReasons()...),
		})
	}
	if err := s.svc.Complete(ctx, req.GetJobId(), report, files); err != nil {
		return nil, err
	}
	return &jobsv1.CompleteJobResponse{}, nil
}

func (s *Server) FailJob(ctx context.Context, req *jobsv1.FailJobRequest) (*jobsv1.FailJobResponse, error) {
	if err := s.requireWorker(ctx); err != nil {
		return nil, err
	}
	if err := s.svc.Fail(ctx, req.GetJobId(), req.GetErrorMessage()); err != nil {
		return nil, err
	}
	return &jobsv1.FailJobResponse{}, nil
}

func (s *Server) SubmitEdits(ctx context.Context, req *jobsv1.SubmitEditsRequest) (*jobsv1.SubmitEditsResponse, error) {
	actor, err := s.actorFrom(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	edits := make([]domain.Edit, 0, len(req.GetEdits()))
	for _, edit := range req.GetEdits() {
		edits = append(edits, domain.Edit{RowKey: edit.GetRowKey(), Field: edit.GetField(), Value: edit.GetValue(), Comment: edit.GetComment()})
	}
	job, err := s.svc.SubmitEdits(ctx, actor, req.GetJobId(), edits)
	if err != nil {
		return nil, err
	}
	return &jobsv1.SubmitEditsResponse{Job: protoJob(job)}, nil
}

func (s *Server) GetReport(ctx context.Context, req *jobsv1.GetReportRequest) (*jobsv1.GetReportResponse, error) {
	actor, err := s.actorFrom(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	report, err := s.svc.GetReport(ctx, actor, req.GetJobId())
	if err != nil {
		return nil, err
	}
	return &jobsv1.GetReportResponse{
		Summary: protoSummary(report.Summary),
		Rows:    protoReportRows(report.Rows),
	}, nil
}

func (s *Server) ListFiles(ctx context.Context, req *jobsv1.ListFilesRequest) (*jobsv1.ListFilesResponse, error) {
	actor, err := s.actorFrom(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	files, err := s.svc.ListFiles(ctx, actor, req.GetJobId())
	if err != nil {
		return nil, err
	}
	out := make([]*jobsv1.FileRef, 0, len(files))
	for _, file := range files {
		out = append(out, &jobsv1.FileRef{
			Id: file.ID, JobId: file.JobID, Kind: file.Kind, ObjectKey: file.ObjectKey,
			Name: file.Name, ContentType: file.ContentType,
		})
	}
	return &jobsv1.ListFilesResponse{Files: out}, nil
}

func (s *Server) UpdateProgress(ctx context.Context, req *jobsv1.UpdateProgressRequest) (*jobsv1.UpdateProgressResponse, error) {
	if err := s.requireWorker(ctx); err != nil {
		return nil, err
	}
	if err := s.svc.UpdateProgress(ctx, req.GetJobId(), domain.Status(req.GetStatus()), req.GetMessage(), req.GetProgress()); err != nil {
		return nil, err
	}
	return &jobsv1.UpdateProgressResponse{}, nil
}

func protoSummary(s domain.ReportSummary) *jobsv1.ReportSummary {
	return &jobsv1.ReportSummary{
		NeedsDecision:     int32Clamp(s.NeedsDecision),
		NotInSource:       int32Clamp(s.NotInSource),
		CheckNameOrVolume: int32Clamp(s.CheckNameOrVolume),
		NotInBlank:        int32Clamp(s.NotInBlank),
		ToOrder:           int32Clamp(s.ToOrder),
		OrderNotNeeded:    int32Clamp(s.OrderNotNeeded),
	}
}

func int32Clamp(n int) int32 {
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	if n < math.MinInt32 {
		return math.MinInt32
	}
	return int32(n)
}

func protoReportRows(rows []domain.ReportRow) []*jobsv1.ReportRow {
	out := make([]*jobsv1.ReportRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &jobsv1.ReportRow{
			Id: row.ID, Category: protoCategory(row.Category),
			Article: row.Article, Name: row.Name, Reasons: row.Reasons,
		})
	}
	return out
}

func protoCategory(c domain.ReportCategory) commonv1.ReportCategory {
	switch c {
	case domain.CategoryNeedsDecision:
		return commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION
	case domain.CategoryNotInSource:
		return commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE
	case domain.CategoryCheckNameOrVolume:
		return commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME
	case domain.CategoryNotInBlank:
		return commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK
	case domain.CategoryToOrder:
		return commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER
	case domain.CategoryOrderNotNeeded:
		return commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED
	default:
		return commonv1.ReportCategory_REPORT_CATEGORY_UNSPECIFIED
	}
}

func reportCategoryName(c commonv1.ReportCategory) string {
	switch c {
	case commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION:
		return string(domain.CategoryNeedsDecision)
	case commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE:
		return string(domain.CategoryNotInSource)
	case commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME:
		return string(domain.CategoryCheckNameOrVolume)
	case commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK:
		return string(domain.CategoryNotInBlank)
	case commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER:
		return string(domain.CategoryToOrder)
	case commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED:
		return string(domain.CategoryOrderNotNeeded)
	default:
		return ""
	}
}
