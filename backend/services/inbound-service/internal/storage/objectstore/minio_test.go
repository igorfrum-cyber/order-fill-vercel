package objectstore

import (
	"errors"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"

	"order-fill/backend/services/inbound-service/internal/domain"
)

func TestParseEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		ssl     bool
		want    string
		secure  bool
		wantErr bool
	}{
		{name: "empty", input: " ", wantErr: true},
		{name: "bare host preserves configured ssl", input: " minio:9000/ ", ssl: true, want: "minio:9000", secure: true},
		{name: "http overrides ssl", input: "http://minio:9000/", ssl: true, want: "minio:9000"},
		{name: "https enables ssl", input: "https://minio:9000", want: "minio:9000", secure: true},
		{name: "unsupported scheme", input: "s3://minio:9000", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, secure, err := parseEndpoint(tt.input, tt.ssl)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseEndpoint(%q) expected an error", tt.input)
				}
				return
			}
			if err != nil || got != tt.want || secure != tt.secure {
				t.Fatalf("parseEndpoint(%q) = %q, %t, %v", tt.input, got, secure, err)
			}
		})
	}
}

func TestNewRequiresEndpointAndBucket(t *testing.T) {
	t.Parallel()

	if _, err := New("", "access", "secret", "bucket", false); err == nil {
		t.Fatal("New accepted an empty endpoint")
	}
	if _, err := New("minio:9000", "access", "secret", " ", false); err == nil {
		t.Fatal("New accepted an empty bucket")
	}
	store, err := New("minio:9000", "access", "secret", "inbound", false)
	if err != nil || store == nil {
		t.Fatalf("New(valid) = %v, %v", store, err)
	}
}

func TestS3TransportWithoutCustomCA(t *testing.T) {
	t.Setenv("GRPC_TLS_CA_FILE", "")
	transport, err := s3Transport()
	if err != nil || transport != nil {
		t.Fatalf("s3Transport() = %v, %v", transport, err)
	}
}

func TestS3TransportRejectsMissingAndInvalidCustomCA(t *testing.T) {
	t.Setenv("GRPC_TLS_CA_FILE", t.TempDir()+"/missing.pem")
	if _, err := s3Transport(); err == nil {
		t.Fatal("s3Transport accepted a missing CA file")
	}

	badCA := t.TempDir() + "/bad.pem"
	if err := os.WriteFile(badCA, []byte("not a PEM certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRPC_TLS_CA_FILE", badCA)
	if _, err := s3Transport(); err == nil {
		t.Fatal("s3Transport accepted invalid PEM data")
	}
}

func TestWrapGetErrorMapsMissingKeys(t *testing.T) {
	t.Parallel()

	for _, code := range []string{"NoSuchKey", "NoSuchBucket", "NotFound"} {
		if err := wrapGetError("key", minio.ErrorResponse{Code: code}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("wrapGetError(%q) = %v, want ErrNotFound", code, err)
		}
	}
	if err := wrapGetError("key", minio.ErrorResponse{StatusCode: 404}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("404 error = %v", err)
	}
	want := errors.New("network failure")
	if err := wrapGetError("key", want); !errors.Is(err, want) || errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unexpected wrapped server error = %v", err)
	}
}
