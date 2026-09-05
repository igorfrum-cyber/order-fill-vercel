package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) ResetUserAccess(ctx context.Context, req *identityv1.ResetUserAccessRequest) (*identityv1.ResetUserAccessResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	token, err := s.users.ResetAccess(ctx, actor, req.GetUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.ResetUserAccessResponse{InviteToken: token}, nil
}
