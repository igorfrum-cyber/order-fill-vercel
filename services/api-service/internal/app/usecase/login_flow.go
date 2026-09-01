package usecase

import (
	"context"
	"fmt"
	"strings"

	"order-fill/services/api-service/internal/domain/identity"
)

func (a *Auth) verifyLogin(ctx context.Context, login string, password string) (identity.User, error) {
	user, err := a.store.GetUserByLogin(ctx, strings.TrimSpace(login))
	if err != nil || user.Disabled() || user.PasswordHash == "" {
		_ = identity.VerifyPassword(identity.DummyPasswordHash(), password)
		return identity.User{}, identity.ErrUnauthorized
	}
	if err := identity.VerifyPassword(user.PasswordHash, password); err != nil {
		return identity.User{}, identity.ErrUnauthorized
	}
	return user, nil
}

func (a *Auth) issueSession(ctx context.Context, user identity.User) (Session, error) {
	raw, err := identity.NewSecret()
	if err != nil {
		return Session{}, err
	}
	expires := a.now().Add(sessionTTL)
	if err := a.store.CreateSession(ctx, identity.HashSecret(raw), user.ID, expires); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{RawToken: raw, User: user, ExpiresAt: expires}, nil
}
