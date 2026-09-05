package grpcapi

import (
	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
	"order-fill/backend/services/twofa-service/internal/service/twofa"
)

type Server struct {
	twofav1.UnimplementedTwoFAServiceServer
	svc *twofa.Service
}

func NewServer(svc *twofa.Service) *Server {
	return &Server{svc: svc}
}

func New(handler twofav1.TwoFAServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		twofav1.RegisterTwoFAServiceServer(s, handler)
	}
	return s
}
