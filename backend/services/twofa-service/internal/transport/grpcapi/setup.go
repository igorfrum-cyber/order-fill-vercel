package grpcapi

import (
	"context"

	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
)

func (s *Server) Setup(ctx context.Context, req *twofav1.SetupRequest) (*twofav1.SetupResponse, error) {
	account := req.GetAccountName()
	if account == "" {
		account = req.GetActorUserId()
	}
	setup, err := s.svc.Setup(ctx, req.GetActorUserId(), account)
	if err != nil {
		return nil, toStatus(err)
	}
	return &twofav1.SetupResponse{OtpauthUrl: setup.OtpauthURL, Secret: setup.Secret, QrPng: setup.QRPNG}, nil
}
