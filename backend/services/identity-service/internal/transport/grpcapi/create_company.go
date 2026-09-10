package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) CreateCompany(ctx context.Context, req *identityv1.CreateCompanyRequest) (*identityv1.CreateCompanyResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	company, err := s.companies.Create(ctx, actor, req.GetName(), req.GetLoginSlug(), domainMatchingMode(req.GetMatchingMode()))
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.CreateCompanyResponse{Company: protoCompany(company)}, nil
}
