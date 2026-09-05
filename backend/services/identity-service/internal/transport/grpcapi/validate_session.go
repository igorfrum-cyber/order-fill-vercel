package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) ValidateSession(ctx context.Context, req *identityv1.ValidateSessionRequest) (*identityv1.ValidateSessionResponse, error) {
	user, err := s.auth.ValidateSession(ctx, req.GetSessionToken())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.ValidateSessionResponse{User: protoUser(user)}, nil
}
