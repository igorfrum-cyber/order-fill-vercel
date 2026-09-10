package twofa_test

import (
	"errors"
	"testing"
	"time"

	"order-fill/backend/services/twofa-service/internal/domain"
	"order-fill/backend/services/twofa-service/internal/ratelimit"
	"order-fill/backend/services/twofa-service/internal/secret"
	"order-fill/backend/services/twofa-service/internal/service/twofa"
	"order-fill/backend/services/twofa-service/internal/storage/memory"
	"order-fill/backend/services/twofa-service/internal/totp"
)

func testService(t *testing.T) (*twofa.Service, *memory.Store, func() time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	box, err := secret.NewBox(make([]byte, secret.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	store := memory.NewStore()
	return twofa.New(store, box, ratelimit.NewRedis(clock), clock), store, clock
}

func TestSetupEnableDisable(t *testing.T) {
	t.Parallel()
	svc, store, clock := testService(t)
	ctx := t.Context()

	setup, err := svc.Setup(ctx, "u1", "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if len(setup.QRPNG) == 0 {
		t.Fatal("qr png is required")
	}
	stored, err := store.Get(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Secret == setup.Secret {
		t.Fatal("secret must be encrypted at rest")
	}

	code, err := totp.CurrentTOTPCode(setup.Secret, clock())
	if err != nil {
		t.Fatal(err)
	}
	codes, err := svc.Enable(ctx, "u1", code)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 8 {
		t.Fatalf("recovery codes=%d", len(codes))
	}
	enabled, err := svc.IsEnabled(ctx, "u1")
	if err != nil || !enabled {
		t.Fatalf("enabled=%v err=%v", enabled, err)
	}

	if err := svc.Disable(ctx, "u1", ""); err != nil {
		t.Fatal(err)
	}
	enabled, err = svc.IsEnabled(ctx, "u1")
	if err != nil || enabled {
		t.Fatalf("after disable enabled=%v err=%v", enabled, err)
	}
}

func TestRecoveryCodeConsumedOnce(t *testing.T) {
	t.Parallel()
	svc, _, clock := testService(t)
	ctx := t.Context()
	setup, err := svc.Setup(ctx, "u1", "buyer")
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.CurrentTOTPCode(setup.Secret, clock())
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := svc.Enable(ctx, "u1", code)
	if err != nil {
		t.Fatal(err)
	}
	used, err := svc.Verify(ctx, "u1", recovery[0])
	if err != nil || !used {
		t.Fatalf("used=%v err=%v", used, err)
	}
	if _, err := svc.Verify(ctx, "u1", recovery[0]); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("second recovery: %v", err)
	}
}

func TestWrongCodeLockout(t *testing.T) {
	t.Parallel()
	svc, _, clock := testService(t)
	ctx := t.Context()
	setup, err := svc.Setup(ctx, "u1", "buyer")
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.CurrentTOTPCode(setup.Secret, clock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Enable(ctx, "u1", code); err != nil {
		t.Fatal(err)
	}
	for range ratelimit.DefaultMax {
		if _, err := svc.Verify(ctx, "u1", "000000"); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("wrong code: %v", err)
		}
	}
	if _, err := svc.Verify(ctx, "u1", code); !errors.Is(err, domain.ErrLocked) {
		t.Fatalf("expected lockout, got %v", err)
	}
}
