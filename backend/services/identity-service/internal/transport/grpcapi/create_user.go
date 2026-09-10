package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/identity-service/internal/domain"
)

func (s *Server) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.CreateUserResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	role, err := domain.ParseRole(req.GetRole())
	if err != nil {
		return nil, toStatus(err)
	}
	user, token, err := s.users.Create(ctx, actor, req.GetCompanyId(), req.GetLogin(), role)
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.CreateUserResponse{User: protoUser(user), InviteToken: token}, nil
}
