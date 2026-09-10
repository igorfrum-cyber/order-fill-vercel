package grpcapi

import (
	"context"

	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
)

func (s *Server) Verify(ctx context.Context, req *twofav1.VerifyRequest) (*twofav1.VerifyResponse, error) {
	used, err := s.svc.Verify(ctx, req.GetUserId(), req.GetCode())
	if err != nil {
		return nil, toStatus(err)
	}
	return &twofav1.VerifyResponse{Ok: true, UsedRecoveryCode: used}, nil
}
