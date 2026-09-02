package identity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// fallbackPasskeyLabel is the user-visible name when the device did not send one.
func fallbackPasskeyLabel() string {
	return "Ключ доступа"
}

const (
	PasskeyPurposeRegister = "register"
	PasskeyPurposeLogin    = "login"
)

// PasskeyCredential is a public authenticator credential. Private keys never
// leave the user's device or password manager.
type PasskeyCredential struct {
	ID         string
	UserID     string
	Name       string
	PublicKey  []byte
	SignCount  uint32
	Transports []string
	AAGUID     []byte
	Raw        []byte
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

func (c PasskeyCredential) DisplayName() string {
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return fallbackPasskeyLabel()
	}
	return name
}

// PasskeyPublicView is the account-settings payload. It must not include
// public keys, attestation, or the stored credential blob.
type PasskeyPublicView struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

func (c PasskeyCredential) PublicView() PasskeyPublicView {
	return PasskeyPublicView{
		ID:         c.ID,
		Name:       c.DisplayName(),
		CreatedAt:  c.CreatedAt.UTC(),
		LastUsedAt: c.LastUsedAt,
	}
}

// PasskeyChallenge is a short-lived WebAuthn ceremony. Session is library JSON,
// not a private key.
type PasskeyChallenge struct {
	ID        string
	UserID    string
	Purpose   string
	Session   []byte
	ExpiresAt time.Time
}

// AssertPasskeyCredentialJSON rejects blobs that look like they contain a
// private key or other secret material.
func AssertPasskeyCredentialJSON(raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("passkey credential json is empty")
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("passkey credential json: %w", err)
	}
	return rejectPrivateKeyMaterial(payload)
}

func rejectPrivateKeyMaterial(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			lowered := strings.ToLower(strings.TrimSpace(key))
			if lowered == "privatekey" || lowered == "private_key" || lowered == "secret" || strings.Contains(lowered, "private") {
				return fmt.Errorf("passkey credential must not contain %s", key)
			}
			if err := rejectPrivateKeyMaterial(nested); err != nil {
				return err
			}
		}
	case []any:
		for _, nested := range typed {
			if err := rejectPrivateKeyMaterial(nested); err != nil {
				return err
			}
		}
	}
	return nil
}
