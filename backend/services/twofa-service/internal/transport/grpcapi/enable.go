package grpcapi

import (
	"context"

	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
)

func (s *Server) Enable(ctx context.Context, req *twofav1.EnableRequest) (*twofav1.EnableResponse, error) {
	codes, err := s.svc.Enable(ctx, req.GetActorUserId(), req.GetCode())
	if err != nil {
		return nil, toStatus(err)
	}
	return &twofav1.EnableResponse{RecoveryCodes: codes}, nil
}
