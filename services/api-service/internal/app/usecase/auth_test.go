package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func TestLoginRejectsUnknownUser(t *testing.T) {
	auth := newTestAuth(t)
	if _, err := auth.Login(context.Background(), "missing", "correct-horse"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestNeedsTwoFactorNudgeFollowsWorkLoginPolicy(t *testing.T) {
	if !NeedsTwoFactorNudge(identity.User{Role: identity.RolePurchaser}) {
		t.Fatal("purchaser without extra login should be nudged")
	}
	if !NeedsTwoFactorNudge(identity.User{Role: identity.RoleCompanyOwner}) {
		t.Fatal("owner without extra login should be nudged")
	}
	if NeedsTwoFactorNudge(identity.User{Role: identity.RolePurchaser, HasPasskey: true}) {
		t.Fatal("a passkey should not be nudged")
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
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if result.TwoFactorRequired || result.ChallengeID != "" {
		t.Fatal("password-only login must not require two-factor")
	}
	if result.Session.RawToken == "" {
		t.Fatal("expected raw token")
	}
	got, err := auth.SessionUser(context.Background(), identity.HashSecret(result.Session.RawToken))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != user.ID {
		t.Fatalf("got user %s", got.ID)
	}
}

func TestListSessionsMarksCurrentFirstAndCanRevokeOther(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	mac := identity.WithClient(context.Background(), identity.ClientInfo{
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15",
		IP:        "10.0.0.2",
	})
	if _, err := auth.Login(mac, user.Login, "correct-horse"); err != nil {
		t.Fatal(err)
	}
	windows := identity.WithClient(context.Background(), identity.ClientInfo{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/128.0.0.0 Safari/537.36",
		IP:        "10.0.0.3",
	})
	second, err := auth.Login(windows, user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	currentHash := identity.HashSecret(second.Session.RawToken)
	items, err := auth.ListSessions(context.Background(), user, currentHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d sessions", len(items))
	}
	if !items[0].Current || items[0].Device != "Chrome на Windows" {
		t.Fatalf("current session %#v", items[0])
	}
	if items[1].Current || items[1].Device != "Safari на Mac" {
		t.Fatalf("other session %#v", items[1])
	}
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "10.0.0.") {
		t.Fatalf("session list leaked IP: %s", raw)
	}
	current, err := auth.RevokeSession(context.Background(), user, items[1].ID, currentHash)
	if err != nil {
		t.Fatal(err)
	}
	if current {
		t.Fatal("revoking the other session must not sign out here")
	}
	left, err := auth.ListSessions(context.Background(), user, currentHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || !left[0].Current {
		t.Fatalf("left %#v", left)
	}
}

func TestLoginWithTOTPDoesNotIssueSession(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedTwoFactorUser(t, store, "buyer", "correct-horse")
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if !result.TwoFactorRequired || result.ChallengeID == "" {
		t.Fatal("expected two-factor challenge")
	}
	if result.Session.RawToken != "" {
		t.Fatal("2FA login must not issue a session after password")
	}
	if len(store.sessions) != 0 {
		t.Fatalf("password step issued a session: %d", len(store.sessions))
	}
	if hasAudit(store, "login_success") {
		t.Fatal("password step must not record login_success")
	}
}

func TestCompleteTwoFactorIssuesSession(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user, settings := seedTwoFactorUserWithSettings(t, store, "buyer", "correct-horse")
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	code, err := identity.CurrentTOTPCode(settings.Secret, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	session, err := auth.CompleteTwoFactor(context.Background(), result.ChallengeID, code)
	if err != nil {
		t.Fatal(err)
	}
	got, err := auth.SessionUser(context.Background(), identity.HashSecret(session.RawToken))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != user.ID {
		t.Fatalf("got user %s", got.ID)
	}
	assertAudit(t, store, "login_success", user.ID, user.CompanyID)
}

func TestCompleteTwoFactorChallengeIsSingleUse(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user, settings := seedTwoFactorUserWithSettings(t, store, "buyer", "correct-horse")
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	code, err := identity.CurrentTOTPCode(settings.Secret, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.CompleteTwoFactor(context.Background(), result.ChallengeID, code); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.CompleteTwoFactor(context.Background(), result.ChallengeID, code); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("reuse: %v", err)
	}
}

func TestCompleteTwoFactorRejectsExpiredChallenge(t *testing.T) {
	store := newMemoryIdentity()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	current := now
	auth := NewAuth(store, func() string { return "id-1" }, func() time.Time { return current })
	user, settings := seedTwoFactorUserWithSettings(t, store, "buyer", "correct-horse")
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	code, err := identity.CurrentTOTPCode(settings.Secret, now)
	if err != nil {
		t.Fatal(err)
	}
	current = now.Add(6 * time.Minute)
	if _, err := auth.CompleteTwoFactor(context.Background(), result.ChallengeID, code); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("expired: %v", err)
	}
}

func TestCompleteTwoFactorWrongCodeIsUnauthorized(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedTwoFactorUser(t, store, "buyer", "correct-horse")
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.CompleteTwoFactor(context.Background(), result.ChallengeID, "000000"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if len(store.sessions) != 0 {
		t.Fatalf("wrong code issued a session: %d", len(store.sessions))
	}
}

func TestCompleteTwoFactorAcceptsRecoveryCodeOnce(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	secret, err := identity.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	raw, hashes, err := identity.GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatal(err)
	}
	enabled := time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)
	if err := store.SaveTOTPSetup(context.Background(), identity.TOTP{
		UserID: user.ID, Secret: secret, EnabledAt: &enabled, RecoveryCodeHashes: hashes,
	}); err != nil {
		t.Fatal(err)
	}
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	session, err := auth.CompleteTwoFactor(context.Background(), result.ChallengeID, raw[0])
	if err != nil {
		t.Fatal(err)
	}
	if session.RawToken == "" {
		t.Fatal("expected session")
	}
	result, err = auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.CompleteTwoFactor(context.Background(), result.ChallengeID, raw[0]); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("reused recovery code: %v", err)
	}
}

func TestVerifyLoginReturnsUserWithoutSession(t *testing.T) {
	auth, store := newTestAuthStore(t)
	seeded := seedPurchaser(t, store, "buyer", "correct-horse")
	user, err := auth.verifyLogin(context.Background(), seeded.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != seeded.ID {
		t.Fatalf("got user %s", user.ID)
	}
	if len(store.sessions) != 0 {
		t.Fatalf("verifyLogin issued a session: %d", len(store.sessions))
	}
}

func TestVerifyLoginRejectsWrongPassword(t *testing.T) {
	auth, store := newTestAuthStore(t)
	seedPurchaser(t, store, "buyer", "correct-horse")
	if _, err := auth.verifyLogin(context.Background(), "buyer", "wrong-password-xx"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if len(store.sessions) != 0 {
		t.Fatalf("failed verifyLogin issued a session: %d", len(store.sessions))
	}
}

func TestIssueSessionCreatesLookupableToken(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	session, err := auth.issueSession(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	got, err := auth.SessionUser(context.Background(), identity.HashSecret(session.RawToken))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != user.ID {
		t.Fatalf("got user %s", got.ID)
	}
	if len(store.sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(store.sessions))
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
	result, err := auth.Login(context.Background(), "buyer", "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	admin := identity.User{ID: "admin-1", CompanyID: user.CompanyID, Role: identity.RoleCompanyAdmin}
	if _, err := auth.ResetAccess(context.Background(), admin, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.SessionUser(context.Background(), identity.HashSecret(result.Session.RawToken)); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("session still valid: %v", err)
	}
}

func TestLogoutEverywhereInvalidatesAllSessions(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	first, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	second, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}

	if err := auth.LogoutEverywhere(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	if _, err := auth.SessionUser(context.Background(), identity.HashSecret(first.Session.RawToken)); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("first session still valid: %v", err)
	}
	if _, err := auth.SessionUser(context.Background(), identity.HashSecret(second.Session.RawToken)); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("second session still valid: %v", err)
	}
	assertAudit(t, store, "logout_everywhere", user.ID, user.CompanyID)
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

func TestLoginRecordsSuccessAudit(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	if _, err := auth.Login(context.Background(), user.Login, "correct-horse"); err != nil {
		t.Fatal(err)
	}
	assertAudit(t, store, "login_success", user.ID, user.CompanyID)
}

func TestLoginFailureDoesNotRecordSuccess(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	if _, err := auth.Login(context.Background(), user.Login, "wrong-password-xx"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if hasAudit(store, "login_success") {
		t.Fatal("failed login must not record login_success")
	}
}

func TestLogoutRecordsAudit(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	result, err := auth.Login(context.Background(), user.Login, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Logout(context.Background(), identity.HashSecret(result.Session.RawToken)); err != nil {
		t.Fatal(err)
	}
	assertAudit(t, store, "logout", user.ID, user.CompanyID)
}

func TestChangePasswordRecordsAudit(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	if err := auth.ChangePassword(context.Background(), user, "correct-horse", "new-password-1"); err != nil {
		t.Fatal(err)
	}
	assertAudit(t, store, "password_changed", user.ID, user.CompanyID)
}

func TestResetAccessRecordsAudit(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	admin := identity.User{ID: "admin-1", CompanyID: user.CompanyID, Role: identity.RoleCompanyAdmin}
	if _, err := auth.ResetAccess(context.Background(), admin, user.ID); err != nil {
		t.Fatal(err)
	}
	assertAudit(t, store, "access_reset", admin.ID, user.CompanyID)
}

func TestCreateUserRecordsInviteCreated(t *testing.T) {
	_, store := newTestAuthStore(t)
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	actor := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if err := store.CreateCompany(context.Background(), identity.Company{ID: "co-2", Name: "Beta"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := admin.CreateUser(context.Background(), actor, "co-2", "admin-2", identity.RoleCompanyAdmin); err != nil {
		t.Fatal(err)
	}
	assertAudit(t, store, "invite_created", actor.ID, "co-2")
}

func TestCreateUserInviteRolesFollowActor(t *testing.T) {
	_, store := newTestAuthStore(t)
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	if err := store.CreateCompany(context.Background(), identity.Company{ID: "co-2", Name: "Beta"}); err != nil {
		t.Fatal(err)
	}
	platform := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if _, _, err := admin.CreateUser(context.Background(), platform, "co-2", "owner-2", identity.RoleCompanyOwner); err != nil {
		t.Fatalf("platform owner invite: %v", err)
	}
	if _, _, err := admin.CreateUser(context.Background(), platform, "co-2", "admin-2", identity.RoleCompanyAdmin); err != nil {
		t.Fatalf("platform admin invite: %v", err)
	}
	if _, _, err := admin.CreateUser(context.Background(), platform, "co-2", "buyer-2", identity.RolePurchaser); err != nil {
		t.Fatalf("platform purchaser invite: %v", err)
	}

	owner := identity.User{ID: "owner-1", CompanyID: "co-2", Role: identity.RoleCompanyOwner}
	if _, _, err := admin.CreateUser(context.Background(), owner, "co-2", "admin-3", identity.RoleCompanyAdmin); err != nil {
		t.Fatalf("owner admin invite: %v", err)
	}
	if _, _, err := admin.CreateUser(context.Background(), owner, "co-2", "buyer-3", identity.RolePurchaser); err != nil {
		t.Fatalf("owner purchaser invite: %v", err)
	}
	if _, _, err := admin.CreateUser(context.Background(), owner, "co-2", "owner-3", identity.RoleCompanyOwner); !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("owner cannot invite owner: got %v", err)
	}

	companyAdmin := identity.User{ID: "admin-1", CompanyID: "co-2", Role: identity.RoleCompanyAdmin}
	if _, _, err := admin.CreateUser(context.Background(), companyAdmin, "co-2", "buyer-4", identity.RolePurchaser); err != nil {
		t.Fatalf("admin purchaser invite: %v", err)
	}
	if _, _, err := admin.CreateUser(context.Background(), companyAdmin, "co-2", "admin-4", identity.RoleCompanyAdmin); !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("admin cannot invite admin: got %v", err)
	}
}

func TestDisableUserCompanyAdminCannotDisableOwner(t *testing.T) {
	_, store := newTestAuthStore(t)
	ctx := context.Background()
	if err := store.CreateCompany(ctx, identity.Company{ID: "co-1", Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	owner := identity.User{ID: "owner-1", CompanyID: "co-1", Role: identity.RoleCompanyOwner, Login: "owner"}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatal(err)
	}
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	actor := identity.User{ID: "admin-1", CompanyID: "co-1", Role: identity.RoleCompanyAdmin}
	if err := admin.DisableUser(ctx, actor, owner.ID); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("company admin disable owner: got %v", err)
	}
}

func TestDisableUserPlatformCanDisableOwner(t *testing.T) {
	_, store := newTestAuthStore(t)
	ctx := context.Background()
	if err := store.CreateCompany(ctx, identity.Company{ID: "co-1", Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	owner := identity.User{ID: "owner-1", CompanyID: "co-1", Role: identity.RoleCompanyOwner, Login: "owner"}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatal(err)
	}
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	actor := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if err := admin.DisableUser(ctx, actor, owner.ID); err != nil {
		t.Fatal(err)
	}
}

func TestResetAccessCompanyAdminCannotResetOwner(t *testing.T) {
	auth, store := newTestAuthStore(t)
	ctx := context.Background()
	if err := store.CreateCompany(ctx, identity.Company{ID: "co-1", Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	owner := identity.User{ID: "owner-1", CompanyID: "co-1", Role: identity.RoleCompanyOwner, Login: "owner"}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatal(err)
	}
	actor := identity.User{ID: "admin-1", CompanyID: "co-1", Role: identity.RoleCompanyAdmin}
	if _, err := auth.ResetAccess(ctx, actor, owner.ID); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("company admin reset owner: got %v", err)
	}
}

func TestDisableUserRecordsAudit(t *testing.T) {
	_, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	actor := identity.User{ID: "admin-1", CompanyID: user.CompanyID, Role: identity.RoleCompanyAdmin}
	if err := admin.DisableUser(context.Background(), actor, user.ID); err != nil {
		t.Fatal(err)
	}
	assertAudit(t, store, "user_disabled", actor.ID, user.CompanyID)
}

func TestDisableCompanyRecordsAudit(t *testing.T) {
	_, store := newTestAuthStore(t)
	if err := store.CreateCompany(context.Background(), identity.Company{ID: "co-1", Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time {
		return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	})
	actor := identity.User{ID: "root", Role: identity.RolePlatformAdmin}
	if err := admin.DisableCompany(context.Background(), actor, "co-1"); err != nil {
		t.Fatal(err)
	}
	assertAudit(t, store, "company_disabled", actor.ID, "co-1")
}

func TestListAuditDropsLoginsAndKeepsAccessEvents(t *testing.T) {
	_, store := newTestAuthStore(t)
	store.users["admin-1"] = identity.User{ID: "admin-1", Login: "root"}
	store.companies["co-1"] = identity.Company{ID: "co-1", Name: "Сияние"}
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	_ = store.InsertAudit(context.Background(), port.AuditEvent{ID: "1", At: now, ActorID: "admin-1", Action: port.AuditLoginSuccess, CompanyID: "co-1"})
	_ = store.InsertAudit(context.Background(), port.AuditEvent{ID: "2", At: now.Add(time.Minute), ActorID: "admin-1", Action: port.AuditInviteCreated, CompanyID: "co-1"})
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time { return now })
	events, err := admin.ListAudit(context.Background(), identity.User{ID: "admin-1", Role: identity.RolePlatformAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != port.AuditInviteCreated {
		t.Fatalf("got %+v", events)
	}
	if events[0].ActorLogin != "root" || events[0].CompanyName != "Сияние" {
		t.Fatalf("missing names %+v", events[0])
	}
}

func TestListUsersAttachesLastLogin(t *testing.T) {
	_, store := newTestAuthStore(t)
	user := identity.User{ID: "u-1", CompanyID: "co-1", Login: "buyer", Role: identity.RolePurchaser}
	store.users[user.ID] = user
	seen := time.Date(2026, 9, 2, 8, 15, 0, 0, time.UTC)
	_ = store.InsertAudit(context.Background(), port.AuditEvent{ID: "login", At: seen, ActorID: user.ID, Action: port.AuditLoginSuccess})
	admin := NewAdmin(store, func() string { return "new-id" }, func() time.Time { return seen })
	items, err := admin.ListUsers(context.Background(), identity.User{ID: "owner", CompanyID: "co-1", Role: identity.RoleCompanyOwner}, "co-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].LastSeenAt == nil || !items[0].LastSeenAt.Equal(seen) {
		t.Fatalf("got %+v", items)
	}
}

func assertAudit(t *testing.T, store *memoryIdentity, action string, actorID string, companyID string) {
	t.Helper()
	for _, event := range store.audits {
		if event.Action != action {
			continue
		}
		if event.ActorID != actorID {
			t.Fatalf("action %s actor: got %q want %q", action, event.ActorID, actorID)
		}
		if event.CompanyID != companyID {
			t.Fatalf("action %s company: got %q want %q", action, event.CompanyID, companyID)
		}
		return
	}
	t.Fatalf("missing audit action %s in %+v", action, store.audits)
}

func hasAudit(store *memoryIdentity, action string) bool {
	for _, event := range store.audits {
		if event.Action == action {
			return true
		}
	}
	return false
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

func seedTwoFactorUser(t *testing.T, store *memoryIdentity, login string, password string) identity.User {
	t.Helper()
	user, _ := seedTwoFactorUserWithSettings(t, store, login, password)
	return user
}

func seedTwoFactorUserWithSettings(t *testing.T, store *memoryIdentity, login string, password string) (identity.User, identity.TOTP) {
	t.Helper()
	user := seedPurchaser(t, store, login, password)
	secret, err := identity.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	_, hashes, err := identity.GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatal(err)
	}
	enabled := time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)
	settings := identity.TOTP{
		UserID:             user.ID,
		Secret:             secret,
		EnabledAt:          &enabled,
		RecoveryCodeHashes: hashes,
	}
	if err := store.SaveTOTPSetup(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	return user, settings
}

type memoryIdentity struct {
	companies         map[string]identity.Company
	users             map[string]identity.User
	sessions          map[string]memorySession
	invites           map[string]memoryInvite
	totp              map[string]identity.TOTP
	challenges        map[string]memoryInvite
	passkeys          map[string]identity.PasskeyCredential
	passkeyChallenges map[string]identity.PasskeyChallenge
	audits            []port.AuditEvent
}

type memorySession struct {
	id        string
	userID    string
	userAgent string
	ip        string
	createdAt time.Time
	expiresAt time.Time
}

type memoryInvite struct {
	userID    string
	expiresAt time.Time
}

func newMemoryIdentity() *memoryIdentity {
	return &memoryIdentity{
		companies:         map[string]identity.Company{},
		users:             map[string]identity.User{},
		sessions:          map[string]memorySession{},
		invites:           map[string]memoryInvite{},
		totp:              map[string]identity.TOTP{},
		challenges:        map[string]memoryInvite{},
		passkeys:          map[string]identity.PasskeyCredential{},
		passkeyChallenges: map[string]identity.PasskeyChallenge{},
	}
}

func (m *memoryIdentity) CountUsers(context.Context) (int, error) { return len(m.users), nil }

func (m *memoryIdentity) CreateCompany(_ context.Context, company identity.Company) error {
	if _, ok := m.companies[company.ID]; ok {
		return identity.ErrConflict
	}
	if company.LoginSlug != "" {
		for _, existing := range m.companies {
			if existing.LoginSlug == company.LoginSlug {
				return identity.ErrConflict
			}
		}
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

func (m *memoryIdentity) GetCompanyByLoginSlug(_ context.Context, slug string) (identity.Company, error) {
	for _, company := range m.companies {
		if company.LoginSlug == slug {
			return company, nil
		}
	}
	return identity.Company{}, identity.ErrNotFound
}

func (m *memoryIdentity) SetCompanyLoginSlug(_ context.Context, id string, slug string) error {
	company, ok := m.companies[id]
	if !ok {
		return identity.ErrNotFound
	}
	for _, existing := range m.companies {
		if existing.ID != id && existing.LoginSlug == slug {
			return identity.ErrConflict
		}
	}
	company.LoginSlug = slug
	m.companies[id] = company
	return nil
}

func (m *memoryIdentity) SetCompanyProfile(_ context.Context, id string, name string, slug string) error {
	company, ok := m.companies[id]
	if !ok {
		return identity.ErrNotFound
	}
	for _, existing := range m.companies {
		if existing.ID != id && existing.LoginSlug == slug {
			return identity.ErrConflict
		}
	}
	company.Name = name
	company.LoginSlug = slug
	m.companies[id] = company
	return nil
}

func (m *memoryIdentity) SetCompanyLogoType(_ context.Context, id string, contentType string) error {
	company, ok := m.companies[id]
	if !ok {
		return identity.ErrNotFound
	}
	company.LogoContentType = contentType
	m.companies[id] = company
	return nil
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
	if user.CompanyID != "" {
		if company, ok := m.companies[user.CompanyID]; ok {
			user.CompanyName = company.Name
			user.CompanyLoginSlug = company.LoginSlug
			user.CompanyHasLogo = company.HasLogo()
			user.CompanyDisabled = company.Disabled()
		}
	}
	if settings, ok := m.totp[user.ID]; ok {
		user.TwoFactorEnabled = settings.Enabled()
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

func (m *memoryIdentity) CreateSession(_ context.Context, session identity.LoginSession) error {
	m.sessions[session.TokenHash] = memorySession{
		id:        session.ID,
		userID:    session.UserID,
		userAgent: session.UserAgent,
		ip:        session.IP,
		createdAt: session.CreatedAt,
		expiresAt: session.ExpiresAt,
	}
	return nil
}

func (m *memoryIdentity) GetSessionUser(_ context.Context, tokenHash string, now time.Time) (identity.User, error) {
	session, ok := m.sessions[tokenHash]
	if !ok || !session.expiresAt.After(now) {
		return identity.User{}, identity.ErrUnauthorized
	}
	return m.GetUserByID(context.Background(), session.userID)
}

func (m *memoryIdentity) ListSessions(_ context.Context, userID string, now time.Time) ([]identity.LoginSession, error) {
	out := make([]identity.LoginSession, 0)
	for hash, session := range m.sessions {
		if session.userID != userID || !session.expiresAt.After(now) {
			continue
		}
		out = append(out, identity.LoginSession{
			ID:        session.id,
			TokenHash: hash,
			UserID:    session.userID,
			UserAgent: session.userAgent,
			IP:        session.ip,
			CreatedAt: session.createdAt,
			ExpiresAt: session.expiresAt,
		})
	}
	return out, nil
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

func (m *memoryIdentity) SaveTOTPSetup(_ context.Context, settings identity.TOTP) error {
	if settings.RecoveryCodeHashes == nil {
		settings.RecoveryCodeHashes = []string{}
	}
	cloned := settings
	cloned.RecoveryCodeHashes = append([]string{}, settings.RecoveryCodeHashes...)
	m.totp[settings.UserID] = cloned
	return nil
}

func (m *memoryIdentity) GetTOTP(_ context.Context, userID string) (identity.TOTP, error) {
	settings, ok := m.totp[userID]
	if !ok {
		return identity.TOTP{}, identity.ErrNotFound
	}
	cloned := settings
	cloned.RecoveryCodeHashes = append([]string{}, settings.RecoveryCodeHashes...)
	return cloned, nil
}

func (m *memoryIdentity) EnableTOTP(_ context.Context, userID string, at time.Time) error {
	settings, ok := m.totp[userID]
	if !ok {
		return identity.ErrNotFound
	}
	settings.EnabledAt = &at
	m.totp[userID] = settings
	return nil
}

func (m *memoryIdentity) DisableTOTP(_ context.Context, userID string) error {
	if _, ok := m.totp[userID]; !ok {
		return identity.ErrNotFound
	}
	delete(m.totp, userID)
	return nil
}

func (m *memoryIdentity) ReplaceRecoveryCodes(_ context.Context, userID string, hashes []string) error {
	settings, ok := m.totp[userID]
	if !ok {
		return identity.ErrNotFound
	}
	if hashes == nil {
		hashes = []string{}
	}
	settings.RecoveryCodeHashes = append([]string{}, hashes...)
	m.totp[userID] = settings
	return nil
}

func (m *memoryIdentity) CreateLoginChallenge(_ context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	m.challenges[tokenHash] = memoryInvite{userID: userID, expiresAt: expiresAt}
	return nil
}

func (m *memoryIdentity) GetLoginChallenge(_ context.Context, tokenHash string, now time.Time) (string, error) {
	challenge, ok := m.challenges[tokenHash]
	if !ok || !challenge.expiresAt.After(now) {
		return "", identity.ErrUnauthorized
	}
	return challenge.userID, nil
}

func (m *memoryIdentity) ConsumeLoginChallenge(_ context.Context, tokenHash string, now time.Time) (string, error) {
	challenge, ok := m.challenges[tokenHash]
	if !ok || !challenge.expiresAt.After(now) {
		return "", identity.ErrUnauthorized
	}
	delete(m.challenges, tokenHash)
	return challenge.userID, nil
}

func (m *memoryIdentity) SavePasskey(_ context.Context, credential identity.PasskeyCredential) error {
	if err := identity.AssertPasskeyCredentialJSON(credential.Raw); err != nil {
		return err
	}
	if _, ok := m.passkeys[credential.ID]; ok {
		return identity.ErrConflict
	}
	cloned := credential
	cloned.Raw = append([]byte{}, credential.Raw...)
	cloned.PublicKey = append([]byte{}, credential.PublicKey...)
	m.passkeys[credential.ID] = cloned
	return nil
}

func (m *memoryIdentity) ListPasskeys(_ context.Context, userID string) ([]identity.PasskeyCredential, error) {
	out := make([]identity.PasskeyCredential, 0)
	for _, credential := range m.passkeys {
		if credential.UserID == userID {
			out = append(out, credential)
		}
	}
	return out, nil
}

func (m *memoryIdentity) GetPasskey(_ context.Context, id string) (identity.PasskeyCredential, error) {
	credential, ok := m.passkeys[id]
	if !ok {
		return identity.PasskeyCredential{}, identity.ErrNotFound
	}
	return credential, nil
}

func (m *memoryIdentity) DeletePasskey(_ context.Context, userID string, id string) error {
	credential, ok := m.passkeys[id]
	if !ok || credential.UserID != userID {
		return identity.ErrNotFound
	}
	delete(m.passkeys, id)
	return nil
}

func (m *memoryIdentity) UpdatePasskey(_ context.Context, credential identity.PasskeyCredential) error {
	if err := identity.AssertPasskeyCredentialJSON(credential.Raw); err != nil {
		return err
	}
	existing, ok := m.passkeys[credential.ID]
	if !ok || existing.UserID != credential.UserID {
		return identity.ErrNotFound
	}
	m.passkeys[credential.ID] = credential
	return nil
}

func (m *memoryIdentity) SavePasskeyChallenge(_ context.Context, challenge identity.PasskeyChallenge) error {
	cloned := challenge
	cloned.Session = append([]byte{}, challenge.Session...)
	m.passkeyChallenges[challenge.ID] = cloned
	return nil
}

func (m *memoryIdentity) GetPasskeyChallenge(_ context.Context, id string, now time.Time) (identity.PasskeyChallenge, error) {
	challenge, ok := m.passkeyChallenges[id]
	if !ok || !challenge.ExpiresAt.After(now) {
		return identity.PasskeyChallenge{}, identity.ErrUnauthorized
	}
	return challenge, nil
}

func (m *memoryIdentity) ConsumePasskeyChallenge(_ context.Context, id string, now time.Time) (identity.PasskeyChallenge, error) {
	challenge, ok := m.passkeyChallenges[id]
	if !ok || !challenge.ExpiresAt.After(now) {
		return identity.PasskeyChallenge{}, identity.ErrUnauthorized
	}
	delete(m.passkeyChallenges, id)
	return challenge, nil
}

func (m *memoryIdentity) InsertAudit(_ context.Context, event port.AuditEvent) error {
	m.audits = append(m.audits, event)
	return nil
}

func (m *memoryIdentity) ListAudit(_ context.Context, limit int, actions []string) ([]port.AuditEvent, error) {
	filtered := make([]port.AuditEvent, 0, len(m.audits))
	allow := map[string]bool{}
	for _, action := range actions {
		allow[action] = true
	}
	for _, event := range m.audits {
		if len(allow) > 0 && !allow[event.Action] {
			continue
		}
		if event.ActorLogin == "" {
			if user, ok := m.users[event.ActorID]; ok {
				event.ActorLogin = user.Login
			}
		}
		if event.CompanyName == "" && event.CompanyID != "" {
			if company, ok := m.companies[event.CompanyID]; ok {
				event.CompanyName = company.Name
			}
		}
		filtered = append(filtered, event)
	}
	if limit <= 0 || limit > len(filtered) {
		limit = len(filtered)
	}
	start := len(filtered) - limit
	if start < 0 {
		start = 0
	}
	out := make([]port.AuditEvent, limit)
	copy(out, filtered[start:])
	return out, nil
}

func (m *memoryIdentity) LastLogins(_ context.Context, userIDs []string) (map[string]time.Time, error) {
	wanted := map[string]bool{}
	for _, id := range userIDs {
		wanted[id] = true
	}
	out := map[string]time.Time{}
	for _, event := range m.audits {
		if event.Action != port.AuditLoginSuccess || !wanted[event.ActorID] {
			continue
		}
		if current, ok := out[event.ActorID]; !ok || event.At.After(current) {
			out[event.ActorID] = event.At.UTC()
		}
	}
	return out, nil
}
