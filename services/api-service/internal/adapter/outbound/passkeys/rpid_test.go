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
	if got != "localhost" {
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

func TestResolveRPIDRejectsMismatchedConfiguredID(t *testing.T) {
	if _, err := resolveRPID("http://127.0.0.1:3200", "localhost"); err == nil {
		t.Fatal("expected mismatch")
	}
}
