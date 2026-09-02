package passkeys

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"order-fill/services/api-service/internal/domain/identity"
)

func TestBeginRegistrationUsesConfiguredParentRPIDOnCompanyDomain(t *testing.T) {
	rp := New("Order Fill", "example.com")
	user := identity.User{ID: "user-1", Login: "buyer"}
	creation, _, err := rp.BeginRegistration("https://kristail.example.com", user, nil)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := creationPublicKey(t, creation)
	rpID, _ := nestedString(publicKey, "rp", "id")
	if rpID != "example.com" {
		t.Fatalf("rp.id = %q", rpID)
	}
}

func TestBeginRegistrationUsesHostRPIDAndBase64UserID(t *testing.T) {
	rp := New("Order Fill", "")
	user := identity.User{ID: "user-1", Login: "buyer"}
	creation, _, err := rp.BeginRegistration("http://christyle.localhost:3200", user, nil)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := creationPublicKey(t, creation)
	rpID, _ := nestedString(publicKey, "rp", "id")
	if rpID != "christyle.localhost" {
		t.Fatalf("rp.id = %q", rpID)
	}
	userID, _ := nestedString(publicKey, "user", "id")
	if userID == "user-1" {
		t.Fatal("user.id must be base64url, not a raw string")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(userID)
	if err != nil {
		t.Fatalf("user.id %q is not base64url: %v", userID, err)
	}
	if string(decoded) != "user-1" {
		t.Fatalf("decoded user.id = %q", decoded)
	}
}

func creationPublicKey(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	publicKey, _ := payload["publicKey"].(map[string]any)
	if publicKey == nil {
		t.Fatalf("missing publicKey: %s", raw)
	}
	return publicKey
}

func nestedString(root map[string]any, keys ...string) (string, bool) {
	current := any(root)
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current = object[key]
	}
	value, ok := current.(string)
	return value, ok
}
