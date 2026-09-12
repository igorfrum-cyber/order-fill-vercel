package grpcapi

import (
	"context"
	"strings"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) GetMe(ctx context.Context, req *identityv1.GetMeRequest) (*identityv1.GetMeResponse, error) {
	if token := strings.TrimSpace(req.GetSessionToken()); token != "" {
		user, err := s.auth.ValidateSession(ctx, token)
		if err != nil {
			return nil, toStatus(err)
		}
		return &identityv1.GetMeResponse{User: protoUser(user)}, nil
	}
	user, err := s.actor(ctx, metaActorID(req.GetMeta()))
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.GetMeResponse{User: protoUser(user)}, nil
}
