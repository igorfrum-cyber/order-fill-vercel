package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) AcceptInvite(ctx context.Context, req *identityv1.AcceptInviteRequest) (*identityv1.AcceptInviteResponse, error) {
	session, err := s.auth.AcceptInvite(ctx, req.GetToken(), req.GetPassword())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.AcceptInviteResponse{
		User:    protoUser(session.User),
		Session: protoSession(session),
	}, nil
}
