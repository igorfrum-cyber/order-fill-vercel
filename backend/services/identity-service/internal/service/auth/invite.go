package auth

import (
	"context"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/password"
)

func (a *Auth) AcceptInvite(ctx context.Context, rawToken, secret string) (Session, error) {
	userID, err := a.store.ConsumeInvite(ctx, hashSecret(rawToken), a.now())
	if err != nil {
		return Session{}, domain.ErrUnauthorized
	}
	hash, err := password.HashPassword(secret)
	if err != nil {
		return Session{}, err
	}
	if err := a.store.SetPasswordHash(ctx, userID, hash); err != nil {
		return Session{}, err
	}
	if err := a.store.DeleteSessionsForUser(ctx, userID); err != nil {
		return Session{}, err
	}
	user, err := a.store.GetUserByID(ctx, userID)
	if err != nil || user.Disabled() {
		return Session{}, domain.ErrUnauthorized
	}
	return a.issueSession(ctx, user)
}

func (a *Auth) ChangePassword(ctx context.Context, actor domain.User, current, next string) error {
	user, err := a.store.GetUserByID(ctx, actor.ID)
	if err != nil || user.Disabled() || user.PasswordHash == "" {
		_ = password.VerifyPassword(password.DummyPasswordHash(), current)
		return domain.ErrUnauthorized
	}
	if err := password.VerifyPassword(user.PasswordHash, current); err != nil {
		return domain.ErrUnauthorized
	}
	hash, err := password.HashPassword(next)
	if err != nil {
		return err
	}
	return a.store.SetPasswordHash(ctx, user.ID, hash)
}

func (a *Auth) FinishPasskeyLogin(ctx context.Context, challengeID, origin string, credential []byte) (Session, error) {
	if a.passkey == nil {
		return Session{}, domain.ErrUnauthorized
	}
	userID, err := a.passkey.FinishLogin(ctx, challengeID, origin, credential)
	if err != nil || userID == "" {
		return Session{}, domain.ErrUnauthorized
	}
	user, err := a.store.GetUserByID(ctx, userID)
	if err != nil || user.Disabled() {
		return Session{}, domain.ErrUnauthorized
	}
	return a.issueSession(ctx, user)
}
