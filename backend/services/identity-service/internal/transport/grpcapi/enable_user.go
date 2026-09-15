package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) EnableUser(ctx context.Context, req *identityv1.EnableUserRequest) (*identityv1.EnableUserResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.users.Enable(ctx, actor, req.GetUserId()); err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.EnableUserResponse{}, nil
}
