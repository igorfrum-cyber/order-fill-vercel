package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) PublicCompany(ctx context.Context, req *identityv1.PublicCompanyRequest) (*identityv1.PublicCompanyResponse, error) {
	company, err := s.companies.PublicBySlug(ctx, req.GetLoginSlug())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.PublicCompanyResponse{Company: protoCompany(company)}, nil
}
