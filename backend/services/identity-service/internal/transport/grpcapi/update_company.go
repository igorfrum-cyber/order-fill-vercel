package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) UpdateCompany(ctx context.Context, req *identityv1.UpdateCompanyRequest) (*identityv1.UpdateCompanyResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	mode := domainMatchingMode(req.GetMatchingMode())
	if req.GetMatchingMode() == 0 {
		mode = ""
	}
	company, err := s.companies.Update(ctx, actor, req.GetCompanyId(), req.GetName(), req.GetLoginSlug(), mode)
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.UpdateCompanyResponse{Company: protoCompany(company)}, nil
}
