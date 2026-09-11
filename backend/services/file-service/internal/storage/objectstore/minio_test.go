package objectstore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseEndpoint(t *testing.T) {
	t.Parallel()
	host, secure, err := parseEndpoint("http://minio:9000", true)
	if err != nil || host != "minio:9000" || secure {
		t.Fatalf("host=%q secure=%v err=%v", host, secure, err)
	}
	host, secure, err = parseEndpoint("minio:9000", false)
	if err != nil || host != "minio:9000" || secure {
		t.Fatalf("bare host=%q secure=%v err=%v", host, secure, err)
	}
}

func TestNewMinIOLoadsCustomCA(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, testCACertPEM(t), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRPC_TLS_CA_FILE", caFile)
	store, err := NewMinIO("minio:9000", "access", "secret", "order-fill", true)
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("expected client")
	}
}

func TestNewMinIORejectsInvalidCA(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, []byte("not-a-cert"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRPC_TLS_CA_FILE", caFile)
	if _, err := NewMinIO("minio:9000", "access", "secret", "order-fill", true); err == nil {
		t.Fatal("expected invalid CA error")
	}
}

func testCACertPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "order-fill-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
