package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) UpdateCompanyOrderProfile(ctx context.Context, req *identityv1.UpdateCompanyOrderProfileRequest) (*identityv1.UpdateCompanyOrderProfileResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	company, err := s.companies.UpdateOrderProfile(ctx, actor, req.GetCompanyId(), domainOrderProfile(req.GetOrderProfile()))
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.UpdateCompanyOrderProfileResponse{Company: protoCompany(company)}, nil
}
