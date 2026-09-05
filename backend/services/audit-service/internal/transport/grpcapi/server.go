package grpcapi

import (
	"context"
	"time"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	auditv1 "order-fill/backend/proto/gen/go/orderfill/audit/v1"
	"order-fill/backend/services/audit-service/internal/service/audit"
)

type Server struct {
	auditv1.UnimplementedAuditServiceServer
	svc *audit.Service
}

func NewServer(svc *audit.Service) *Server { return &Server{svc: svc} }

func New(handler auditv1.AuditServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		auditv1.RegisterAuditServiceServer(s, handler)
	}
	return s
}

func (s *Server) Record(ctx context.Context, req *auditv1.RecordRequest) (*auditv1.RecordResponse, error) {
	actor, company := "", ""
	if req.GetMeta() != nil {
		actor = req.GetMeta().GetActorUserId()
		company = req.GetMeta().GetCompanyId()
	}
	id, err := s.svc.Record(req.GetType(), actor, company, req.GetJobId(), req.GetPayloadJson())
	if err != nil {
		return nil, err
	}
	return &auditv1.RecordResponse{Id: id}, nil
}

func (s *Server) ListEvents(_ context.Context, req *auditv1.ListEventsRequest) (*auditv1.ListEventsResponse, error) {
	company := req.GetCompanyId()
	if company == "" && req.GetMeta() != nil {
		company = req.GetMeta().GetCompanyId()
	}
	events := s.svc.List(company)
	out := make([]*auditv1.Event, 0, len(events))
	for _, e := range events {
		out = append(out, &auditv1.Event{
			Id: e.ID, Type: e.Type, ActorUserId: e.ActorID, CompanyId: e.CompanyID,
			JobId: e.JobID, CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339), PayloadJson: e.Payload,
		})
	}
	return &auditv1.ListEventsResponse{Events: out}, nil
}
