package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

func TestLoginRejectsUnknownUser(t *testing.T) {
	auth := newTestAuth(t)
	if _, err := auth.Login(context.Background(), "missing", "correct-horse"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	if _, err := auth.Login(context.Background(), user.Login, "wrong-password-xx"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestLoginIssuesSession(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	session, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if session.RawToken == "" {
		t.Fatal("expected raw token")
	}
	got, err := auth.SessionUser(context.Background(), identity.HashSecret(session.RawToken))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != user.ID {
		t.Fatalf("got user %s", got.ID)
	}
}

func TestInviteIsSingleUse(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := identity.User{ID: "user-1", CompanyID: "co-1", Login: "buyer", Role: identity.RolePurchaser}
	if err := store.CreateCompany(context.Background(), identity.Company{ID: "co-1", Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	raw, err := auth.createInvite(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.AcceptInvite(context.Background(), raw, "correct-horse"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.AcceptInvite(context.Background(), raw, "correct-horse"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("reuse: %v", err)
	}
}

func TestBootstrapOnlyWhenEmpty(t *testing.T) {
	auth, _ := newTestAuthStore(t)
	raw, created, err := auth.Bootstrap(context.Background(), "root")
	if err != nil || !created || raw == "" {
		t.Fatalf("first bootstrap: created=%v raw=%q err=%v", created, raw, err)
	}
	again, created, err := auth.Bootstrap(context.Background(), "root")
	if err != nil || created || again != "" {
		t.Fatalf("second bootstrap: created=%v raw=%q err=%v", created, again, err)
	}
}

func TestResetAccessInvalidatesSessions(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	session, err := auth.Login(context.Background(), "buyer", "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	admin := identity.User{ID: "admin-1", CompanyID: user.CompanyID, Role: identity.RoleCompanyAdmin}
	if _, err := auth.ResetAccess(context.Background(), admin, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.SessionUser(context.Background(), identity.HashSecret(session.RawToken)); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("session still valid: %v", err)
	}
}

func TestChangePasswordRejectsWrongCurrent(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	if err := auth.ChangePassword(context.Background(), user, "wrong-password-xx", "new-password-1"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestChangePasswordUpdatesHash(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	if err := auth.ChangePassword(context.Background(), user, "correct-horse", "new-password-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Login(context.Background(), "buyer", "correct-horse"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatal("old password still worked")
	}
	if _, err := auth.Login(context.Background(), "buyer", "new-password-1"); err != nil {
		t.Fatal(err)
	}
}

func newTestAuth(t *testing.T) *Auth {
	t.Helper()
	auth, _ := newTestAuthStore(t)
	return auth
}

func newTestAuthStore(t *testing.T) (*Auth, *memoryIdentity) {
	t.Helper()
	store := newMemoryIdentity()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	auth := NewAuth(store, func() string { return "id-1" }, func() time.Time { return now })
	return auth, store
}

func seedPurchaser(t *testing.T, store *memoryIdentity, login string, password string) identity.User {
	t.Helper()
	ctx := context.Background()
	if err := store.CreateCompany(ctx, identity.Company{ID: "co-1", Name: "Acme"}); err != nil && !errors.Is(err, identity.ErrConflict) {
		t.Fatal(err)
	}
	hash, err := identity.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	user := identity.User{
		ID: "user-" + login, CompanyID: "co-1", Login: login, PasswordHash: hash, Role: identity.RolePurchaser,
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	return user
}

type memoryIdentity struct {
	companies map[string]identity.Company
	users     map[string]identity.User
	sessions  map[string]memorySession
	invites   map[string]memoryInvite
	audits    []port.AuditEvent
}

type memorySession struct {
	userID    string
	expiresAt time.Time
}

type memoryInvite struct {
	userID    string
	expiresAt time.Time
}

func newMemoryIdentity() *memoryIdentity {
	return &memoryIdentity{
		companies: map[string]identity.Company{},
		users:     map[string]identity.User{},
		sessions:  map[string]memorySession{},
		invites:   map[string]memoryInvite{},
	}
}

func (m *memoryIdentity) CountUsers(context.Context) (int, error) { return len(m.users), nil }

func (m *memoryIdentity) CreateCompany(_ context.Context, company identity.Company) error {
	if _, ok := m.companies[company.ID]; ok {
		return identity.ErrConflict
	}
	m.companies[company.ID] = company
	return nil
}

func (m *memoryIdentity) GetCompany(_ context.Context, id string) (identity.Company, error) {
	company, ok := m.companies[id]
	if !ok {
		return identity.Company{}, identity.ErrNotFound
	}
	return company, nil
}

func (m *memoryIdentity) ListCompanies(context.Context) ([]identity.Company, error) {
	out := make([]identity.Company, 0, len(m.companies))
	for _, company := range m.companies {
		out = append(out, company)
	}
	return out, nil
}

func (m *memoryIdentity) DisableCompany(_ context.Context, id string, at time.Time) error {
	company, ok := m.companies[id]
	if !ok {
		return identity.ErrNotFound
	}
	company.DisabledAt = &at
	m.companies[id] = company
	return nil
}

func (m *memoryIdentity) CreateUser(_ context.Context, user identity.User) error {
	for _, existing := range m.users {
		if existing.Login == user.Login {
			return identity.ErrConflict
		}
	}
	m.users[user.ID] = user
	return nil
}

func (m *memoryIdentity) hydrate(user identity.User) identity.User {
	if user.CompanyID == "" {
		return user
	}
	if company, ok := m.companies[user.CompanyID]; ok {
		user.CompanyDisabled = company.Disabled()
	}
	return user
}

func (m *memoryIdentity) GetUserByID(_ context.Context, id string) (identity.User, error) {
	user, ok := m.users[id]
	if !ok {
		return identity.User{}, identity.ErrNotFound
	}
	return m.hydrate(user), nil
}

func (m *memoryIdentity) GetUserByLogin(_ context.Context, login string) (identity.User, error) {
	for _, user := range m.users {
		if user.Login == login {
			return m.hydrate(user), nil
		}
	}
	return identity.User{}, identity.ErrNotFound
}

func (m *memoryIdentity) ListUsers(_ context.Context, companyID string) ([]identity.User, error) {
	out := make([]identity.User, 0)
	for _, user := range m.users {
		if companyID == "" || user.CompanyID == companyID {
			out = append(out, m.hydrate(user))
		}
	}
	return out, nil
}

func (m *memoryIdentity) SetPasswordHash(_ context.Context, userID string, hash string) error {
	user, ok := m.users[userID]
	if !ok {
		return identity.ErrNotFound
	}
	user.PasswordHash = hash
	m.users[userID] = user
	return nil
}

func (m *memoryIdentity) ClearPasswordHash(_ context.Context, userID string) error {
	return m.SetPasswordHash(context.Background(), userID, "")
}

func (m *memoryIdentity) DisableUser(_ context.Context, id string, at time.Time) error {
	user, ok := m.users[id]
	if !ok {
		return identity.ErrNotFound
	}
	user.DisabledAt = &at
	m.users[id] = user
	return nil
}

func (m *memoryIdentity) CreateSession(_ context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	m.sessions[tokenHash] = memorySession{userID: userID, expiresAt: expiresAt}
	return nil
}

func (m *memoryIdentity) GetSessionUser(_ context.Context, tokenHash string, now time.Time) (identity.User, error) {
	session, ok := m.sessions[tokenHash]
	if !ok || !session.expiresAt.After(now) {
		return identity.User{}, identity.ErrUnauthorized
	}
	return m.GetUserByID(context.Background(), session.userID)
}

func (m *memoryIdentity) DeleteSession(_ context.Context, tokenHash string) error {
	delete(m.sessions, tokenHash)
	return nil
}

func (m *memoryIdentity) DeleteSessionsForUser(_ context.Context, userID string) error {
	for hash, session := range m.sessions {
		if session.userID == userID {
			delete(m.sessions, hash)
		}
	}
	return nil
}

func (m *memoryIdentity) CreateInvite(_ context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	m.invites[tokenHash] = memoryInvite{userID: userID, expiresAt: expiresAt}
	return nil
}

func (m *memoryIdentity) DeleteInvitesForUser(_ context.Context, userID string) error {
	for hash, invite := range m.invites {
		if invite.userID == userID {
			delete(m.invites, hash)
		}
	}
	return nil
}

func (m *memoryIdentity) ConsumeInvite(_ context.Context, tokenHash string, now time.Time) (string, error) {
	invite, ok := m.invites[tokenHash]
	if !ok || !invite.expiresAt.After(now) {
		return "", identity.ErrUnauthorized
	}
	delete(m.invites, tokenHash)
	return invite.userID, nil
}

func (m *memoryIdentity) InsertAudit(_ context.Context, event port.AuditEvent) error {
	m.audits = append(m.audits, event)
	return nil
}

func (m *memoryIdentity) ListAudit(_ context.Context, limit int) ([]port.AuditEvent, error) {
	if limit <= 0 || limit > len(m.audits) {
		limit = len(m.audits)
	}
	out := make([]port.AuditEvent, limit)
	copy(out, m.audits[len(m.audits)-limit:])
	return out, nil
}
