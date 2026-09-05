package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) RevokeSession(ctx context.Context, req *identityv1.RevokeSessionRequest) (*identityv1.RevokeSessionResponse, error) {
	actor, err := s.actor(ctx, req.GetActorUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.auth.RevokeSession(ctx, actor, req.GetSessionId(), req.GetSessionToken()); err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.RevokeSessionResponse{}, nil
}
