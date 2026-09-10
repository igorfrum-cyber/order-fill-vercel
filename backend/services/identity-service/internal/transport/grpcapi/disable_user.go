package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) DisableUser(ctx context.Context, req *identityv1.DisableUserRequest) (*identityv1.DisableUserResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.users.Disable(ctx, actor, req.GetUserId()); err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.DisableUserResponse{}, nil
}
