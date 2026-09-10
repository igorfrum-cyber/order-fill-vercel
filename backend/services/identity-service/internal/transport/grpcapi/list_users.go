package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) ListUsers(ctx context.Context, req *identityv1.ListUsersRequest) (*identityv1.ListUsersResponse, error) {
	actor, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	items, err := s.users.List(ctx, actor, req.GetCompanyId())
	if err != nil {
		return nil, toStatus(err)
	}
	out := make([]*identityv1.User, 0, len(items))
	for _, item := range items {
		out = append(out, protoUser(item))
	}
	return &identityv1.ListUsersResponse{Users: out}, nil
}
