package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/password"
	"order-fill/backend/services/identity-service/internal/service/auth"
	"order-fill/backend/services/identity-service/internal/service/users"
	"order-fill/backend/services/identity-service/internal/storage/memory"
)

const testPassword = "password10"

type fakeTwoFA struct {
	enabled map[string]bool
	code    string
}

func (f *fakeTwoFA) IsEnabled(_ context.Context, userID string) (bool, error) {
	return f.enabled[userID], nil
}

func (f *fakeTwoFA) Verify(_ context.Context, userID, code string) error {
	if !f.enabled[userID] || code != f.code {
		return domain.ErrUnauthorized
	}
	return nil
}

type fakePasskey struct {
	userID         string
	sessionsMinted int
	has            bool
}

func (f *fakePasskey) FinishLogin(_ context.Context, _, _ string, credential []byte) (string, error) {
	if len(credential) == 0 {
		return "", domain.ErrUnauthorized
	}
	return f.userID, nil
}

func (f *fakePasskey) HasCredentials(_ context.Context, userID string) (bool, error) {
	return f.has && userID == f.userID, nil
}

func setup(t *testing.T) (*memory.Store, *auth.Auth, *users.Users, domain.User, *fakeTwoFA, *fakePasskey) {
	t.Helper()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	totp := &fakeTwoFA{enabled: map[string]bool{}, code: "000000"}
	keys := &fakePasskey{}
	svc := auth.New(store, totp, keys, func() time.Time { return now })
	usersSvc := users.New(store, func() time.Time { return now })

	hash, err := password.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	user := domain.User{
		ID:           "user-1",
		Login:        "buyer",
		PasswordHash: hash,
		Role:         domain.RolePurchaser,
		CompanyID:    "co-1",
		CreatedAt:    now,
	}
	if err := store.CreateCompany(t.Context(), domain.Company{
		ID: "co-1", Name: "Acme", LoginSlug: "acme", MatchingMode: domain.MatchingModeStandard, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(t.Context(), user); err != nil {
		t.Fatal(err)
	}
	keys.userID = user.ID
	return store, svc, usersSvc, user, totp, keys
}

func TestLoginMissingUserAndWrongPasswordMatch(t *testing.T) {
	t.Parallel()
	_, svc, _, _, _, _ := setup(t)
	ctx := t.Context()

	_, missing := svc.Login(ctx, "nobody", testPassword)
	_, wrong := svc.Login(ctx, "buyer", "wrong-password-that-is-long")
	if !errors.Is(missing, domain.ErrUnauthorized) || !errors.Is(wrong, domain.ErrUnauthorized) {
		t.Fatalf("missing=%v wrong=%v", missing, wrong)
	}
	if missing.Error() != wrong.Error() {
		t.Fatalf("auth failures must look the same: %q vs %q", missing, wrong)
	}
}

func TestDisabledUserCannotLogin(t *testing.T) {
	t.Parallel()
	store, svc, _, user, _, _ := setup(t)
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	if err := store.DisableUser(t.Context(), user.ID, now); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Login(t.Context(), user.Login, testPassword)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestInviteIsOneTime(t *testing.T) {
	t.Parallel()
	store, svc, usersSvc, _, _, _ := setup(t)
	ctx := t.Context()
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	if err := store.CreateUser(ctx, admin); err != nil {
		t.Fatal(err)
	}
	_, token, err := usersSvc.Create(ctx, admin, "co-1", "newbie", domain.RolePurchaser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AcceptInvite(ctx, token, testPassword); err != nil {
		t.Fatal(err)
	}
	_, err = svc.AcceptInvite(ctx, token, testPassword)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("second consume: %v", err)
	}
}

func TestResetAccessClearsPasswordAndSessions(t *testing.T) {
	t.Parallel()
	store, svc, usersSvc, user, _, _ := setup(t)
	ctx := t.Context()
	if _, err := svc.Login(ctx, user.Login, testPassword); err != nil {
		t.Fatal(err)
	}
	if store.SessionCount() != 1 {
		t.Fatalf("sessions=%d", store.SessionCount())
	}
	admin := domain.User{ID: "admin", Role: domain.RolePlatformAdmin}
	if err := store.CreateUser(ctx, admin); err != nil {
		t.Fatal(err)
	}
	if _, err := usersSvc.ResetAccess(ctx, admin, user.ID); err != nil {
		t.Fatal(err)
	}
	if store.SessionCount() != 0 {
		t.Fatalf("sessions after reset=%d", store.SessionCount())
	}
	_, err := svc.Login(ctx, user.Login, testPassword)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("password should be cleared: %v", err)
	}
}

func TestOnlyIdentityMintsSessions(t *testing.T) {
	t.Parallel()
	store, svc, _, user, _, keys := setup(t)
	session, err := svc.FinishPasskeyLogin(t.Context(), "ch", "http://localhost", []byte("assertion"))
	if err != nil {
		t.Fatal(err)
	}
	if session.RawToken == "" {
		t.Fatal("identity must mint a session token")
	}
	if keys.sessionsMinted != 0 {
		t.Fatal("passkey client must not mint sessions")
	}
	if store.SessionCount() != 1 {
		t.Fatalf("sessions=%d", store.SessionCount())
	}
	got, err := svc.ValidateSession(t.Context(), session.RawToken)
	if err != nil || got.ID != user.ID {
		t.Fatalf("validate: user=%v err=%v", got, err)
	}
}

func TestValidateSessionHydratesSecurityFlags(t *testing.T) {
	t.Parallel()
	_, svc, _, user, totp, keys := setup(t)
	totp.enabled[user.ID] = true
	keys.has = true
	session, err := svc.FinishPasskeyLogin(t.Context(), "ch", "http://localhost", []byte("assertion"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.ValidateSession(t.Context(), session.RawToken)
	if err != nil {
		t.Fatal(err)
	}
	if !got.TwoFactorEnabled || !got.HasPasskey {
		t.Fatalf("twofa=%v passkey=%v", got.TwoFactorEnabled, got.HasPasskey)
	}
}

func TestTOTPEnabledUserGetsChallengeUntilVerified(t *testing.T) {
	t.Parallel()
	store, svc, _, user, totp, _ := setup(t)
	totp.enabled[user.ID] = true
	ctx := t.Context()

	result, err := svc.Login(ctx, user.Login, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !result.TwoFactorRequired || result.ChallengeID == "" || result.Session.RawToken != "" {
		t.Fatalf("expected challenge only, got %+v", result)
	}
	if store.SessionCount() != 0 {
		t.Fatal("session must not be minted before TOTP")
	}
	if _, err := svc.CompleteTwoFactor(ctx, result.ChallengeID, "bad"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("wrong code: %v", err)
	}
	session, err := svc.CompleteTwoFactor(ctx, result.ChallengeID, totp.code)
	if err != nil {
		t.Fatal(err)
	}
	if session.RawToken == "" {
		t.Fatal("session after TOTP")
	}
	if store.SessionCount() != 1 {
		t.Fatalf("sessions=%d", store.SessionCount())
	}
}
