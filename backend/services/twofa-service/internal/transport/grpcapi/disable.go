package grpcapi

import (
	"context"

	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
)

func (s *Server) Disable(ctx context.Context, req *twofav1.DisableRequest) (*twofav1.DisableResponse, error) {
	if err := s.svc.Disable(ctx, req.GetActorUserId(), req.GetCode()); err != nil {
		return nil, toStatus(err)
	}
	return &twofav1.DisableResponse{}, nil
}
