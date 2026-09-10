package grpcapi

import (
	"context"

	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
)

func (s *Server) IsEnabled(ctx context.Context, req *twofav1.IsEnabledRequest) (*twofav1.IsEnabledResponse, error) {
	enabled, err := s.svc.IsEnabled(ctx, req.GetUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &twofav1.IsEnabledResponse{Enabled: enabled}, nil
}
