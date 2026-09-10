package grpcapi

import (
	"context"

	"order-fill/backend/services/identity-service/internal/domain"
)

func (s *Server) actor(ctx context.Context, userID string) (domain.User, error) {
	if userID == "" {
		return domain.User{}, domain.ErrUnauthorized
	}
	user, err := s.auth.GetUser(ctx, userID)
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	if user.Disabled() {
		return domain.User{}, domain.ErrUnauthorized
	}
	return user, nil
}
