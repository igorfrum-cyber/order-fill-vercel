package identity

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPasskeyPublicViewOmitsKeyMaterial(t *testing.T) {
	created := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	credential := PasskeyCredential{
		ID:        "cred-1",
		UserID:    "user-1",
		Name:      "MacBook",
		PublicKey: []byte{0x04, 0x11, 0x22},
		SignCount: 4,
		Raw:       []byte(`{"id":"cred-1","publicKey":"secret-looking"}`),
		CreatedAt: created,
	}

	view := credential.PublicView()
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(encoded)
	if strings.Contains(payload, "publicKey") || strings.Contains(payload, "public_key") {
		t.Fatalf("public view leaked key material: %s", payload)
	}
	if strings.Contains(payload, "secret-looking") || strings.Contains(payload, "\\u0004") {
		t.Fatalf("public view leaked raw credential: %s", payload)
	}
	if view.ID != "cred-1" || view.Name != "MacBook" {
		t.Fatalf("view: %#v", view)
	}
}

func TestPasskeyDisplayNameDefaultsToDeviceLogin(t *testing.T) {
	credential := PasskeyCredential{Name: "  "}
	if got := credential.DisplayName(); got != "Ключ доступа" {
		t.Fatalf("got %q", got)
	}
}

func TestPasskeyStoredShapeHasNoPrivateKey(t *testing.T) {
	raw := []byte(`{"id":"YWJj","publicKey":"BKEi","attestationType":"none"}`)
	if err := AssertPasskeyCredentialJSON(raw); err != nil {
		t.Fatal(err)
	}
	if err := AssertPasskeyCredentialJSON([]byte(`{"privateKey":"aaa"}`)); err == nil {
		t.Fatal("expected private key material to be rejected")
	}
}
