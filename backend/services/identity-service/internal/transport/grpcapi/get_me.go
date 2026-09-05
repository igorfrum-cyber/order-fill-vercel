package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) GetMe(ctx context.Context, req *identityv1.GetMeRequest) (*identityv1.GetMeResponse, error) {
	user, err := s.auth.ValidateSession(ctx, req.GetSessionToken())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.GetMeResponse{User: protoUser(user)}, nil
}
