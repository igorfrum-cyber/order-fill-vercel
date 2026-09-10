package grpcapi

import (
	"context"
	"time"

	passkeyv1 "order-fill/backend/proto/gen/go/orderfill/passkey/v1"
	"order-fill/backend/services/passkey-service/internal/domain"
)

func protoCredential(view domain.PasskeyPublicView) *passkeyv1.Credential {
	out := &passkeyv1.Credential{
		Id:        view.ID,
		Name:      view.Name,
		CreatedAt: view.CreatedAt.UTC().Format(time.RFC3339),
	}
	if view.LastUsedAt != nil {
		out.LastUsedAt = view.LastUsedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (s *Server) BeginRegistration(ctx context.Context, req *passkeyv1.BeginRegistrationRequest) (*passkeyv1.BeginRegistrationResponse, error) {
	begin, err := s.svc.BeginRegistration(ctx, req.GetActorUserId(), req.GetOrigin())
	if err != nil {
		return nil, toStatus(err)
	}
	return &passkeyv1.BeginRegistrationResponse{ChallengeId: begin.ChallengeID, OptionsJson: begin.Options}, nil
}

func (s *Server) FinishRegistration(ctx context.Context, req *passkeyv1.FinishRegistrationRequest) (*passkeyv1.FinishRegistrationResponse, error) {
	view, err := s.svc.FinishRegistration(ctx, req.GetActorUserId(), req.GetOrigin(), req.GetChallengeId(), req.GetCredentialJson())
	if err != nil {
		return nil, toStatus(err)
	}
	return &passkeyv1.FinishRegistrationResponse{Credential: protoCredential(view)}, nil
}

func (s *Server) ListCredentials(ctx context.Context, req *passkeyv1.ListCredentialsRequest) (*passkeyv1.ListCredentialsResponse, error) {
	items, err := s.svc.List(ctx, req.GetActorUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	out := make([]*passkeyv1.Credential, 0, len(items))
	for _, item := range items {
		out = append(out, protoCredential(item))
	}
	return &passkeyv1.ListCredentialsResponse{Credentials: out}, nil
}

func (s *Server) DeleteCredential(ctx context.Context, req *passkeyv1.DeleteCredentialRequest) (*passkeyv1.DeleteCredentialResponse, error) {
	if err := s.svc.Delete(ctx, req.GetActorUserId(), req.GetCredentialId()); err != nil {
		return nil, toStatus(err)
	}
	return &passkeyv1.DeleteCredentialResponse{}, nil
}

func (s *Server) BeginLogin(ctx context.Context, req *passkeyv1.BeginLoginRequest) (*passkeyv1.BeginLoginResponse, error) {
	begin, err := s.svc.BeginLogin(ctx, req.GetLogin(), req.GetOrigin())
	if err != nil {
		return nil, toStatus(err)
	}
	return &passkeyv1.BeginLoginResponse{ChallengeId: begin.ChallengeID, OptionsJson: begin.Options}, nil
}

func (s *Server) FinishLogin(ctx context.Context, req *passkeyv1.FinishLoginRequest) (*passkeyv1.FinishLoginResponse, error) {
	userID, err := s.svc.FinishLogin(ctx, req.GetOrigin(), req.GetChallengeId(), req.GetCredentialJson())
	if err != nil {
		return nil, toStatus(err)
	}
	return &passkeyv1.FinishLoginResponse{UserId: userID}, nil
}
