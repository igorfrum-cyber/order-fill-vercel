package users

import (
	"context"
	"time"

	"order-fill/backend/services/identity-service/internal/domain"
)

const inviteTTL = 72 * time.Hour

type Store interface {
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	CreateUser(ctx context.Context, user domain.User) error
	ListUsers(ctx context.Context, companyID string) ([]domain.User, error)
	DisableUser(ctx context.Context, id string, at time.Time) error
	ClearPasswordHash(ctx context.Context, userID string) error
	DeleteSessionsForUser(ctx context.Context, userID string) error
	DeleteInvitesForUser(ctx context.Context, userID string) error
	CreateInvite(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	GetCompany(ctx context.Context, id string) (domain.Company, error)
}

type Users struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) *Users {
	if now == nil {
		now = time.Now
	}
	return &Users{store: store, now: now}
}

func (u *Users) Get(ctx context.Context, id string) (domain.User, error) {
	return u.store.GetUserByID(ctx, id)
}
