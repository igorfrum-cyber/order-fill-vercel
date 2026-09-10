package secret

import (
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
