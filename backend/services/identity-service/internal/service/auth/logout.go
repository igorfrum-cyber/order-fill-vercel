package auth

import (
	"context"
	"strings"

	"order-fill/backend/services/identity-service/internal/domain"
)

func (a *Auth) Logout(ctx context.Context, rawToken string) error {
	return a.store.DeleteSession(ctx, hashSecret(strings.TrimSpace(rawToken)))
}

func (a *Auth) LogoutEverywhere(ctx context.Context, actor domain.User) error {
	return a.store.DeleteSessionsForUser(ctx, actor.ID)
}

func (a *Auth) ListSessions(ctx context.Context, actor domain.User, rawToken string) ([]domain.SessionPublicView, error) {
	items, err := a.store.ListSessions(ctx, actor.ID, a.now())
	if err != nil {
		return nil, err
	}
	currentHash := hashSecret(strings.TrimSpace(rawToken))
	out := make([]domain.SessionPublicView, 0, len(items))
	var current domain.SessionPublicView
	hasCurrent := false
	for _, item := range items {
		view := item.PublicView(currentHash)
		if view.Current {
			current = view
			hasCurrent = true
			continue
		}
		out = append(out, view)
	}
	if hasCurrent {
		return append([]domain.SessionPublicView{current}, out...), nil
	}
	return out, nil
}

func (a *Auth) RevokeSession(ctx context.Context, actor domain.User, sessionID, _ string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.ErrNotFound
	}
	items, err := a.store.ListSessions(ctx, actor.ID, a.now())
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID != sessionID {
			continue
		}
		return a.store.DeleteSession(ctx, item.TokenHash)
	}
	return domain.ErrNotFound
}
