package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) DisableCompany(ctx context.Context, req *identityv1.DisableCompanyRequest) (*identityv1.DisableCompanyResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.companies.Disable(ctx, actor, req.GetCompanyId()); err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.DisableCompanyResponse{}, nil
}
