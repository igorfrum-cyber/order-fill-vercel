package grpcapi

import (
	"context"
	"time"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	auditv1 "order-fill/backend/proto/gen/go/orderfill/audit/v1"
	"order-fill/backend/services/audit-service/internal/clients/identity"
	"order-fill/backend/services/audit-service/internal/domain"
	"order-fill/backend/services/audit-service/internal/service/audit"
)

const rolePlatformAdmin = "platform_admin"

type Server struct {
	auditv1.UnimplementedAuditServiceServer
	svc    *audit.Service
	actors ActorLookup
}

type ActorLookup interface {
	Actor(ctx context.Context, userID string) (identity.Actor, error)
}

func NewServer(svc *audit.Service, actors ActorLookup) *Server {
	return &Server{svc: svc, actors: actors}
}

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
	id, err := s.svc.Record(ctx, req.GetType(), actor, company, req.GetJobId(), req.GetPayloadJson())
	if err != nil {
		return nil, err
	}
	return &auditv1.RecordResponse{Id: id}, nil
}

func (s *Server) ListEvents(ctx context.Context, req *auditv1.ListEventsRequest) (*auditv1.ListEventsResponse, error) {
	company := req.GetCompanyId()
	if company == "" && req.GetMeta() != nil {
		company = req.GetMeta().GetCompanyId()
	}
	userID := ""
	if req.GetMeta() != nil {
		userID = req.GetMeta().GetActorUserId()
	}
	if s.actors != nil {
		actor, err := s.actors.Actor(ctx, userID)
		if err != nil {
			return nil, err
		}
		if actor.Role != rolePlatformAdmin {
			if actor.CompanyID == "" {
				return nil, domain.ErrUnauthorized
			}
			company = actor.CompanyID
		}
	} else if company == "" {
		return nil, domain.ErrUnauthorized
	}
	events, err := s.svc.List(ctx, company)
	if err != nil {
		return nil, err
	}
	out := make([]*auditv1.Event, 0, len(events))
	for _, e := range events {
		out = append(out, &auditv1.Event{
			Id: e.ID, Type: e.Type, ActorUserId: e.ActorID, CompanyId: e.CompanyID,
			JobId: e.JobID, CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339), PayloadJson: e.Payload,
		})
	}
	return &auditv1.ListEventsResponse{Events: out}, nil
}
