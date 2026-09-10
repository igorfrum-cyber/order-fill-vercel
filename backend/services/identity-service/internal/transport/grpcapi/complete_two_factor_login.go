package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) CompleteTwoFactorLogin(ctx context.Context, req *identityv1.CompleteTwoFactorLoginRequest) (*identityv1.CompleteTwoFactorLoginResponse, error) {
	session, err := s.auth.CompleteTwoFactor(ctx, req.GetChallengeId(), req.GetCode())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.CompleteTwoFactorLoginResponse{
		User:    protoUser(session.User),
		Session: protoSession(session),
	}, nil
}
