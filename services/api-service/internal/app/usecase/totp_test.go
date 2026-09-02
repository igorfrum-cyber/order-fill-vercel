package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"order-fill/services/api-service/internal/domain/identity"
)

func TestStartTOTPSetupStoresPendingSecret(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	setup, err := auth.StartTOTPSetup(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	if setup.Secret == "" || setup.AuthURL == "" || setup.QRPNGBase64 == "" {
		t.Fatalf("incomplete setup %#v", setup)
	}
	stored, err := store.GetTOTP(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Enabled() {
		t.Fatal("setup must not enable totp yet")
	}
	if stored.Secret != setup.Secret {
		t.Fatal("stored secret mismatch")
	}
}

func TestEnableTOTPReturnsRecoveryCodesOnce(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	setup, err := auth.StartTOTPSetup(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	code, err := identity.CurrentTOTPCode(setup.Secret, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := auth.EnableTOTP(context.Background(), user, code)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 8 {
		t.Fatalf("recovery codes: %d", len(raw))
	}
	stored, err := store.GetTOTP(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Enabled() {
		t.Fatal("expected enabled totp")
	}
	for _, code := range raw {
		for _, hash := range stored.RecoveryCodeHashes {
			if hash == code {
				t.Fatal("must not store raw recovery code")
			}
		}
	}
}

func TestEnableTOTPRejectsWrongCode(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	if _, err := auth.StartTOTPSetup(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.EnableTOTP(context.Background(), user, "000000"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestDisableTOTPRequiresPassword(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user, _ := seedTwoFactorUserWithSettings(t, store, "buyer", "correct-horse")
	if err := auth.DisableTOTP(context.Background(), user, "wrong-password-xx"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if err := auth.DisableTOTP(context.Background(), user, "correct-horse"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetTOTP(context.Background(), user.ID); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("expected totp removed: %v", err)
	}
}
