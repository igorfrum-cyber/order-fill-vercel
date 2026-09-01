package usecase

import (
	"context"
	"strings"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

const (
	sessionTTL = 7 * 24 * time.Hour
	inviteTTL  = 72 * time.Hour
)

type Session struct {
	RawToken  string
	User      identity.User
	ExpiresAt time.Time
}

type Auth struct {
	store port.IdentityStore
	newID port.IDGenerator
	now   port.Clock
}

func NewAuth(store port.IdentityStore, newID port.IDGenerator, now port.Clock) *Auth {
	return &Auth{store: store, newID: newID, now: now}
}

func (a *Auth) Login(ctx context.Context, login string, password string) (Session, error) {
	user, err := a.verifyLogin(ctx, login, password)
	if err != nil {
		return Session{}, err
	}
	session, err := a.issueSession(ctx, user)
	if err != nil {
		return Session{}, err
	}
	a.recordAudit(ctx, user, port.AuditLoginSuccess, user.CompanyID, "")
	return session, nil
}

func (a *Auth) AcceptInvite(ctx context.Context, rawToken string, password string) (Session, error) {
	userID, err := a.store.ConsumeInvite(ctx, identity.HashSecret(rawToken), a.now())
	if err != nil {
		return Session{}, identity.ErrUnauthorized
	}
	hash, err := identity.HashPassword(password)
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
	if err != nil {
		return Session{}, err
	}
	if user.Disabled() {
		return Session{}, identity.ErrUnauthorized
	}
	return a.issueSession(ctx, user)
}

func (a *Auth) ChangePassword(ctx context.Context, actor identity.User, current string, next string) error {
	user, err := a.store.GetUserByID(ctx, actor.ID)
	if err != nil || user.Disabled() || user.PasswordHash == "" {
		_ = identity.VerifyPassword(identity.DummyPasswordHash(), current)
		return identity.ErrUnauthorized
	}
	if err := identity.VerifyPassword(user.PasswordHash, current); err != nil {
		return identity.ErrUnauthorized
	}
	hash, err := identity.HashPassword(next)
	if err != nil {
		return err
	}
	if err := a.store.SetPasswordHash(ctx, user.ID, hash); err != nil {
		return err
	}
	a.recordAudit(ctx, user, port.AuditPasswordChanged, user.CompanyID, "")
	return nil
}

func (a *Auth) Logout(ctx context.Context, tokenHash string) error {
	user, lookupErr := a.store.GetSessionUser(ctx, tokenHash, a.now())
	if err := a.store.DeleteSession(ctx, tokenHash); err != nil {
		return err
	}
	if lookupErr == nil {
		a.recordAudit(ctx, user, port.AuditLogout, user.CompanyID, "")
	}
	return nil
}

func (a *Auth) SessionUser(ctx context.Context, tokenHash string) (identity.User, error) {
	user, err := a.store.GetSessionUser(ctx, tokenHash, a.now())
	if err != nil {
		return identity.User{}, identity.ErrUnauthorized
	}
	if user.Disabled() {
		return identity.User{}, identity.ErrUnauthorized
	}
	return user, nil
}

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
	user := identity.User{
		ID:        a.newID(),
		Login:     login,
		Role:      identity.RolePlatformAdmin,
		CreatedAt: a.now(),
	}
	if err := a.store.CreateUser(ctx, user); err != nil {
		return "", false, err
	}
	raw, err := a.createInvite(ctx, user.ID)
	if err != nil {
		return "", false, err
	}
	return raw, true, nil
}

func (a *Auth) ResetAccess(ctx context.Context, actor identity.User, userID string) (string, error) {
	user, err := a.store.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if !canManageUser(actor, user) {
		return "", identity.ErrNotFound
	}
	if err := a.store.ClearPasswordHash(ctx, userID); err != nil {
		return "", err
	}
	if err := a.store.DeleteSessionsForUser(ctx, userID); err != nil {
		return "", err
	}
	if err := a.store.DeleteInvitesForUser(ctx, userID); err != nil {
		return "", err
	}
	raw, err := a.createInvite(ctx, userID)
	if err != nil {
		return "", err
	}
	a.recordAudit(ctx, actor, port.AuditAccessReset, user.CompanyID, "")
	return raw, nil
}

func (a *Auth) createInvite(ctx context.Context, userID string) (string, error) {
	raw, err := identity.NewSecret()
	if err != nil {
		return "", err
	}
	if err := a.store.CreateInvite(ctx, identity.HashSecret(raw), userID, a.now().Add(inviteTTL)); err != nil {
		return "", err
	}
	return raw, nil
}

func (a *Auth) recordAudit(ctx context.Context, actor identity.User, action string, companyID string, jobID string) {
	if a.store == nil {
		return
	}
	_ = a.store.InsertAudit(ctx, port.AuditEvent{
		ID:        a.newID(),
		At:        a.now().UTC(),
		ActorID:   actor.ID,
		Action:    action,
		CompanyID: companyID,
		JobID:     jobID,
	})
}

func canManageUser(actor identity.User, target identity.User) bool {
	if actor.Disabled() {
		return false
	}
	if actor.Role == identity.RolePlatformAdmin {
		return true
	}
	return actor.Role == identity.RoleCompanyAdmin &&
		actor.CompanyID != "" &&
		actor.CompanyID == target.CompanyID &&
		target.Role != identity.RolePlatformAdmin
}
