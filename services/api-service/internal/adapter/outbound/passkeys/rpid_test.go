package passkeys

import "testing"

func TestResolveRPIDUsesConfiguredParentDomain(t *testing.T) {
	got, err := resolveRPID("https://kristail.example.com", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRPIDLocalhostSubdomain(t *testing.T) {
	got, err := resolveRPID("http://acme.localhost:3200", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "acme.localhost" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRPIDBareLocalhost(t *testing.T) {
	got, err := resolveRPID("http://localhost:3200", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "localhost" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRPIDDoesNotUseLocalhostPublicSuffix(t *testing.T) {
	got, err := resolveRPID("http://christyle.localhost:3200", "localhost")
	if err != nil {
		t.Fatal(err)
	}
	if got != "christyle.localhost" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRPIDLoopbackIP(t *testing.T) {
	got, err := resolveRPID("http://127.0.0.1:3200", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRPIDRejectsLANIP(t *testing.T) {
	if _, err := resolveRPID("http://192.168.31.108:3200", ""); err == nil {
		t.Fatal("LAN IP must not be a passkey relying party")
	}
}

func TestResolveRPIDRejectsMismatchedConfiguredID(t *testing.T) {
	if _, err := resolveRPID("http://127.0.0.1:3200", "localhost"); err == nil {
		t.Fatal("expected mismatch")
	}
}
