package secret

import "testing"

func TestRandomTokenNotStoredInPlain(t *testing.T) {
	raw, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 32 {
		t.Fatalf("too short: %d", len(raw))
	}
	sum := HashSecret(raw)
	if sum == raw {
		t.Fatal("must not store raw token")
	}
	if !SecretEqual(sum, HashSecret(raw)) {
		t.Fatal("expected equal hashes")
	}
}

func TestSecretEqualRejectsDifferentHashes(t *testing.T) {
	left, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	if SecretEqual(HashSecret(left), HashSecret(right)) {
		t.Fatal("distinct secrets must not compare equal")
	}
}
