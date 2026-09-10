package auth

import (
	"context"
	"strings"

	"order-fill/backend/services/identity-service/internal/domain"
)

func (a *Auth) ValidateSession(ctx context.Context, rawToken string) (domain.User, error) {
	user, err := a.store.GetSessionUser(ctx, hashSecret(strings.TrimSpace(rawToken)), a.now())
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	if user.Disabled() {
		return domain.User{}, domain.ErrUnauthorized
	}
	if a.twoFA != nil {
		enabled, err := a.twoFA.IsEnabled(ctx, user.ID)
		if err == nil {
			user.TwoFactorEnabled = enabled
		}
	}
	if a.passkey != nil {
		has, err := a.passkey.HasCredentials(ctx, user.ID)
		if err == nil {
			user.HasPasskey = has
		}
	}
	return user, nil
}
