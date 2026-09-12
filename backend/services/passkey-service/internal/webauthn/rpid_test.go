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

func TestResolveRPIDRejectsLoopbackIP(t *testing.T) {
	t.Parallel()
	if _, err := resolveRPID("http://127.0.0.1:3200", ""); err == nil {
		t.Fatal("loopback IP must not be a passkey relying party")
	}
}

func TestResolveRPIDRejectsLANIP(t *testing.T) {
	t.Parallel()
	if _, err := resolveRPID("http://192.168.31.108:3200", ""); err == nil {
		t.Fatal("LAN IP must not be a passkey relying party")
	}
}

func TestResolveRPIDRejectsPlainHTTPPublicHost(t *testing.T) {
	t.Parallel()
	if _, err := resolveRPID("http://example.com", "example.com"); err == nil {
		t.Fatal("plain http public origins must not be allowed")
	}
}
