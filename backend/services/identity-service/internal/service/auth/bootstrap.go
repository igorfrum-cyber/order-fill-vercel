package auth

import (
	"context"
	"strings"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/secret"
)

func (a *Auth) Bootstrap(ctx context.Context, login string) (string, bool, error) {
	count, err := a.store.CountUsers(ctx)
	if err != nil {
		return "", false, err
	}
	if count > 0 {
		return "", false, nil
	}
	login = strings.TrimSpace(login)
	if login == "" {
		login = "admin"
	}
	id, err := secret.NewSecret()
	if err != nil {
		return "", false, err
	}
	user := domain.User{
		ID:        id,
		Login:     login,
		Role:      domain.RolePlatformAdmin,
		CreatedAt: a.now(),
	}
	if err := a.store.CreateUser(ctx, user); err != nil {
		return "", false, err
	}
	raw, err := secret.NewSecret()
	if err != nil {
		return "", false, err
	}
	if err := a.store.CreateInvite(ctx, secret.HashSecret(raw), user.ID, a.now().Add(inviteTTL)); err != nil {
		return "", false, err
	}
	return raw, true, nil
}
