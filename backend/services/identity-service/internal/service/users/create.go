package users

import (
	"context"
	"fmt"
	"strings"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/secret"
)

func (u *Users) Create(ctx context.Context, actor domain.User, companyID, login string, role domain.Role) (domain.User, string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return domain.User{}, "", fmt.Errorf("%w: login is required", domain.ErrInvalid)
	}
	if domain.BoundToOwnCompany(actor) {
		companyID = actor.CompanyID
	}
	if !domain.CanManageCompany(actor, companyID) || companyID == "" {
		return domain.User{}, "", domain.ErrNotFound
	}
	if !domain.CanInviteRole(actor, role) {
		return domain.User{}, "", fmt.Errorf("%w: unsupported role", domain.ErrInvalid)
	}
	if _, err := u.store.GetCompany(ctx, companyID); err != nil {
		return domain.User{}, "", domain.ErrNotFound
	}
	id, err := secret.NewSecret()
	if err != nil {
		return domain.User{}, "", err
	}
	user := domain.User{
		ID:        id,
		CompanyID: companyID,
		Login:     login,
		Role:      role,
		CreatedAt: u.now(),
	}
	if err := u.store.CreateUser(ctx, user); err != nil {
		return domain.User{}, "", err
	}
	raw, err := u.createInvite(ctx, user.ID)
	if err != nil {
		return domain.User{}, "", err
	}
	return user, raw, nil
}

func (u *Users) List(ctx context.Context, actor domain.User, companyID string) ([]domain.User, error) {
	if domain.BoundToOwnCompany(actor) {
		companyID = actor.CompanyID
	}
	if !domain.CanManageCompany(actor, companyID) {
		return nil, domain.ErrNotFound
	}
	return u.store.ListUsers(ctx, companyID)
}

func (u *Users) Disable(ctx context.Context, actor domain.User, userID string) error {
	user, err := u.store.GetUserByID(ctx, userID)
	if err != nil {
		return domain.ErrNotFound
	}
	if !domain.CanManageUser(actor, user) {
		return domain.ErrNotFound
	}
	return u.store.DisableUser(ctx, userID, u.now())
}

func (u *Users) ResetAccess(ctx context.Context, actor domain.User, userID string) (string, error) {
	user, err := u.store.GetUserByID(ctx, userID)
	if err != nil {
		return "", domain.ErrNotFound
	}
	if !domain.CanManageUser(actor, user) {
		return "", domain.ErrNotFound
	}
	if err := u.store.ClearPasswordHash(ctx, userID); err != nil {
		return "", err
	}
	if err := u.store.DeleteSessionsForUser(ctx, userID); err != nil {
		return "", err
	}
	if err := u.store.DeleteInvitesForUser(ctx, userID); err != nil {
		return "", err
	}
	return u.createInvite(ctx, userID)
}

func (u *Users) createInvite(ctx context.Context, userID string) (string, error) {
	raw, err := secret.NewSecret()
	if err != nil {
		return "", err
	}
	if err := u.store.CreateInvite(ctx, secret.HashSecret(raw), userID, u.now().Add(inviteTTL)); err != nil {
		return "", err
	}
	return raw, nil
}
