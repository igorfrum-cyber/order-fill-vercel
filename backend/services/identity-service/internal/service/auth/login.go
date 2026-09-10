package auth

import (
	"context"
	"strings"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/password"
)

func (a *Auth) Login(ctx context.Context, login, secret string) (LoginResult, error) {
	user, err := a.verifyLogin(ctx, login, secret)
	if err != nil {
		return LoginResult{}, err
	}
	if a.twoFA != nil {
		enabled, err := a.twoFA.IsEnabled(ctx, user.ID)
		if err != nil {
			return LoginResult{}, err
		}
		if enabled {
			raw, err := newSecret()
			if err != nil {
				return LoginResult{}, err
			}
			if err := a.store.CreateLoginChallenge(ctx, hashSecret(raw), user.ID, a.now().Add(challengeTTL)); err != nil {
				return LoginResult{}, err
			}
			return LoginResult{TwoFactorRequired: true, ChallengeID: raw}, nil
		}
	}
	session, err := a.issueSession(ctx, user)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Session: session}, nil
}

func (a *Auth) verifyLogin(ctx context.Context, login, secret string) (domain.User, error) {
	user, err := a.store.GetUserByLogin(ctx, strings.TrimSpace(login))
	if err != nil || user.Disabled() || user.PasswordHash == "" {
		_ = password.VerifyPassword(password.DummyPasswordHash(), secret)
		return domain.User{}, domain.ErrUnauthorized
	}
	if err := password.VerifyPassword(user.PasswordHash, secret); err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	return user, nil
}
