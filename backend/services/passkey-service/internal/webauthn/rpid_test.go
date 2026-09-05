package webauthn

import "testing"

func TestResolveRPIDUsesConfiguredParentDomain(t *testing.T) {
	t.Parallel()
	got, err := resolveRPID("https://kristail.example.com", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRPIDRejectsLANIP(t *testing.T) {
	t.Parallel()
	if _, err := resolveRPID("http://192.168.31.108:3200", ""); err == nil {
		t.Fatal("LAN IP must not be a passkey relying party")
	}
}
