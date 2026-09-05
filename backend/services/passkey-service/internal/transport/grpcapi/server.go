package grpcapi

import (
	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	passkeyv1 "order-fill/backend/proto/gen/go/orderfill/passkey/v1"
	"order-fill/backend/services/passkey-service/internal/service/passkey"
)

type Server struct {
	passkeyv1.UnimplementedPasskeyServiceServer
	svc *passkey.Service
}

func NewServer(svc *passkey.Service) *Server {
	return &Server{svc: svc}
}

func New(handler passkeyv1.PasskeyServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		passkeyv1.RegisterPasskeyServiceServer(s, handler)
	}
	return s
}
