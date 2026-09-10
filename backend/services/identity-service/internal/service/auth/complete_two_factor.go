package auth

import (
	"context"
	"strings"

	"order-fill/backend/services/identity-service/internal/domain"
)

func (a *Auth) CompleteTwoFactor(ctx context.Context, challengeID, code string) (Session, error) {
	challengeID = strings.TrimSpace(challengeID)
	if challengeID == "" || a.twoFA == nil {
		return Session{}, domain.ErrUnauthorized
	}
	hash := hashSecret(challengeID)
	userID, err := a.store.GetLoginChallenge(ctx, hash, a.now())
	if err != nil {
		return Session{}, domain.ErrUnauthorized
	}
	user, err := a.store.GetUserByID(ctx, userID)
	if err != nil || user.Disabled() {
		return Session{}, domain.ErrUnauthorized
	}
	if err := a.twoFA.Verify(ctx, user.ID, strings.TrimSpace(code)); err != nil {
		return Session{}, domain.ErrUnauthorized
	}
	if _, err := a.store.ConsumeLoginChallenge(ctx, hash, a.now()); err != nil {
		return Session{}, domain.ErrUnauthorized
	}
	return a.issueSession(ctx, user)
}
