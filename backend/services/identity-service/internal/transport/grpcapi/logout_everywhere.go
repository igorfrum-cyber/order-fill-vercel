package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) LogoutEverywhere(ctx context.Context, req *identityv1.LogoutEverywhereRequest) (*identityv1.LogoutEverywhereResponse, error) {
	actor, err := s.actor(ctx, req.GetActorUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	if err := s.auth.LogoutEverywhere(ctx, actor); err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.LogoutEverywhereResponse{}, nil
}
