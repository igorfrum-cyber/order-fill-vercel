package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) ListCompanies(ctx context.Context, req *identityv1.ListCompaniesRequest) (*identityv1.ListCompaniesResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	items, err := s.companies.List(ctx, actor)
	if err != nil {
		return nil, toStatus(err)
	}
	out := make([]*identityv1.Company, 0, len(items))
	for _, item := range items {
		out = append(out, protoCompany(item))
	}
	return &identityv1.ListCompaniesResponse{Companies: out}, nil
}
