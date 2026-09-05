package grpcapi

import (
	"context"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (s *Server) FinishPasskeyLogin(ctx context.Context, req *identityv1.FinishPasskeyLoginRequest) (*identityv1.FinishPasskeyLoginResponse, error) {
	session, err := s.auth.FinishPasskeyLogin(ctx, req.GetChallengeId(), req.GetOrigin(), req.GetCredentialJson())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.FinishPasskeyLoginResponse{
		User:    protoUser(session.User),
		Session: protoSession(session),
	}, nil
}
