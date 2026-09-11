package secret

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestBoxRoundTrip(t *testing.T) {
	t.Parallel()
	box, err := NewBox(make([]byte, KeySize))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("totp-secret")
	if err != nil {
		t.Fatal(err)
	}
	if sealed == "totp-secret" {
		t.Fatal("must not store plaintext")
	}
	got, err := box.Open(sealed)
	if err != nil || got != "totp-secret" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestBoxFromMasterUsesHKDFNotSHA256(t *testing.T) {
	t.Parallel()
	master := "local-dev-twofa-master-key-32bytes!!"
	box, err := NewBoxFromMaster(master)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := NewBox(sha256Sum(master))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("totp-secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Open(sealed); err == nil {
		t.Fatal("hkdf sealed blob must not open with sha256(master)")
	}
	got, err := box.Open(sealed)
	if err != nil || got != "totp-secret" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestBoxFromMasterOpensLegacySHA256Seals(t *testing.T) {
	t.Parallel()
	master := "local-dev-twofa-master-key-32bytes!!"
	legacy, err := NewBox(sha256Sum(master))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := legacy.Seal("old-secret")
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewBoxFromMaster(master)
	if err != nil {
		t.Fatal(err)
	}
	got, err := box.Open(sealed)
	if err != nil || got != "old-secret" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func sha256Sum(master string) []byte {
	sum := sha256.Sum256([]byte(master))
	out := make([]byte, KeySize)
	copy(out, sum[:])
	return out
}

func TestBoxRejectsTamper(t *testing.T) {
	t.Parallel()
	box, err := NewBox(append(make([]byte, KeySize-1), 1))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("secret")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawStdEncoding.DecodeString(sealed)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0xff
	if _, err := box.Open(base64.RawStdEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("expected auth failure")
	}
}
