package auth

import (
	"context"
	"fmt"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/secret"
)

func (a *Auth) issueSession(ctx context.Context, user domain.User) (Session, error) {
	raw, err := secret.NewSecret()
	if err != nil {
		return Session{}, err
	}
	id, err := secret.NewSecret()
	if err != nil {
		return Session{}, err
	}
	expires := a.now().Add(sessionTTL)
	client := domain.ClientFrom(ctx)
	if err := a.store.CreateSession(ctx, domain.LoginSession{
		ID:        id,
		TokenHash: secret.HashSecret(raw),
		UserID:    user.ID,
		UserAgent: client.UserAgent,
		IP:        client.IP,
		CreatedAt: a.now().UTC(),
		ExpiresAt: expires,
	}); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{ID: id, RawToken: raw, User: user, ExpiresAt: expires}, nil
}

func newSecret() (string, error) {
	return secret.NewSecret()
}

func hashSecret(raw string) string {
	return secret.HashSecret(raw)
}
