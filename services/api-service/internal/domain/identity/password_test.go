package identity

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPassword(hash, "correct-horse"); err != nil {
		t.Fatalf("expected match: %v", err)
	}
	if err := VerifyPassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestRejectShortPassword(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRejectLongPassword(t *testing.T) {
	long := make([]byte, MaxPasswordLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := HashPassword(string(long)); err == nil {
		t.Fatal("expected error")
	}
}
