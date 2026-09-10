package auth

import (
	"context"
	"time"

	"order-fill/backend/services/identity-service/internal/clients/passkey"
	"order-fill/backend/services/identity-service/internal/clients/twofa"
	"order-fill/backend/services/identity-service/internal/domain"
)

const (
	sessionTTL   = 8 * time.Hour
	inviteTTL    = 72 * time.Hour
	challengeTTL = 5 * time.Minute
)

type Store interface {
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (domain.User, error)
	CreateUser(ctx context.Context, user domain.User) error
	CountUsers(ctx context.Context) (int, error)
	SetPasswordHash(ctx context.Context, userID, hash string) error
	CreateSession(ctx context.Context, session domain.LoginSession) error
	GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (domain.User, error)
	ListSessions(ctx context.Context, userID string, now time.Time) ([]domain.LoginSession, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteSessionsForUser(ctx context.Context, userID string) error
	CreateInvite(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	ConsumeInvite(ctx context.Context, tokenHash string, now time.Time) (string, error)
	CreateLoginChallenge(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	GetLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error)
	ConsumeLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error)
}

type Session struct {
	ID        string
	RawToken  string
	User      domain.User
	ExpiresAt time.Time
}

type LoginResult struct {
	Session           Session
	TwoFactorRequired bool
	ChallengeID       string
}

type Auth struct {
	store   Store
	twoFA   twofa.Client
	passkey passkey.Client
	now     func() time.Time
}

func New(store Store, totp twofa.Client, keys passkey.Client, now func() time.Time) *Auth {
	if now == nil {
		now = time.Now
	}
	return &Auth{store: store, twoFA: totp, passkey: keys, now: now}
}

func (a *Auth) GetUser(ctx context.Context, id string) (domain.User, error) {
	return a.store.GetUserByID(ctx, id)
}
