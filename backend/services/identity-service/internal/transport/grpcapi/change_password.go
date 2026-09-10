package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) ChangePassword(ctx context.Context, req *identityv1.ChangePasswordRequest) (*identityv1.ChangePasswordResponse, error) {
	actor, err := s.actor(ctx, req.GetActorUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.auth.ChangePassword(ctx, actor, req.GetCurrentPassword(), req.GetNewPassword()); err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.ChangePasswordResponse{}, nil
}
