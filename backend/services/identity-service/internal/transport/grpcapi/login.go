package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	result, err := s.auth.Login(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		return nil, toStatus(err)
	}
	if result.TwoFactorRequired {
		return &identityv1.LoginResponse{TwoFactorRequired: true, ChallengeId: result.ChallengeID}, nil
	}
	return &identityv1.LoginResponse{
		User:    protoUser(result.Session.User),
		Session: protoSession(result.Session),
	}, nil
}
