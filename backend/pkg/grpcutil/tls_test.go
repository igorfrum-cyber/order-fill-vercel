package grpcutil

import "testing"

func TestGRPCTLSModeDefaultsInsecure(t *testing.T) {
	t.Setenv(envGRPCTLSMode, "")
	mode, err := grpcTLSMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != "insecure" {
		t.Fatalf("mode=%q", mode)
	}
}

func TestGRPCTLSModeRejectsUnknown(t *testing.T) {
	t.Setenv(envGRPCTLSMode, "maybe")
	if _, err := grpcTLSMode(); err == nil {
		t.Fatal("expected error")
	}
}

func TestClientTransportRequiresMTLSMaterial(t *testing.T) {
	t.Setenv(envGRPCTLSMode, "mtls")
	t.Setenv(envGRPCTLSCertFile, "")
	t.Setenv(envGRPCTLSKeyFile, "")
	t.Setenv(envGRPCTLSCAFile, "")
	if _, err := clientTransportCredentials("identity-service:9091"); err == nil {
		t.Fatal("expected missing TLS material error")
	}
}

func TestGRPCTLSServerName(t *testing.T) {
	t.Setenv(envGRPCTLSServerName, "")
	if got := grpcTLSServerName("identity-service:9091"); got != "identity-service" {
		t.Fatalf("server name=%q", got)
	}
	t.Setenv(envGRPCTLSServerName, "internal.order-fill")
	if got := grpcTLSServerName("identity-service:9091"); got != "internal.order-fill" {
		t.Fatalf("explicit server name=%q", got)
	}
}
