package totp

import (
	"strings"
	"testing"
	"time"
)

func TestValidTOTPCodeVerifies(t *testing.T) {
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	code, err := CurrentTOTPCode(secret, at)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTOTP(secret, code, at); err != nil {
		t.Fatalf("expected valid code: %v", err)
	}
}

func TestWrongTOTPCodeFails(t *testing.T) {
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	code, err := CurrentTOTPCode(secret, at)
	if err != nil {
		t.Fatal(err)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	if err := VerifyTOTP(secret, wrong, at); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestRecoveryCodeAcceptedOnce(t *testing.T) {
	raw, hashes, err := GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 8 || len(hashes) != 8 {
		t.Fatalf("got %d raw and %d hashes", len(raw), len(hashes))
	}
	remaining, err := ConsumeRecoveryCode(hashes, raw[0])
	if err != nil {
		t.Fatalf("expected accepted recovery code: %v", err)
	}
	if len(remaining) != 7 {
		t.Fatalf("remaining hashes: %d", len(remaining))
	}
	if _, err := ConsumeRecoveryCode(remaining, raw[0]); err == nil {
		t.Fatal("expected recovery code to work once")
	}
}

func TestRecoveryCodeHashDoesNotStoreRawCode(t *testing.T) {
	raw, hashes, err := GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(hashes, "")
	for _, code := range raw {
		if code == "" {
			t.Fatal("empty recovery code")
		}
		if strings.Contains(joined, code) {
			t.Fatal("must not store raw recovery code")
		}
	}
}

func TestTOTPAuthURLContainsSecret(t *testing.T) {
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := TOTPAuthURL(secret, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(authURL, secret) {
		t.Fatal("otpauth url must include the secret")
	}
	if !strings.Contains(authURL, "otpauth://totp/") {
		t.Fatalf("unexpected url %s", authURL)
	}
	png, err := TOTPQR(secret, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 32 {
		t.Fatalf("qr png too small: %d", len(png))
	}
}
