package grpcapi

import (
	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/identity-service/internal/service/auth"
	"order-fill/backend/services/identity-service/internal/service/companies"
	"order-fill/backend/services/identity-service/internal/service/users"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	auth      *auth.Auth
	users     *users.Users
	companies *companies.Companies
}

func NewServer(a *auth.Auth, u *users.Users, c *companies.Companies) *Server {
	return &Server{auth: a, users: u, companies: c}
}

func New(handler identityv1.IdentityServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		identityv1.RegisterIdentityServiceServer(s, handler)
	}
	return s
}
