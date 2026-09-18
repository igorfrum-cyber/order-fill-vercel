package config

import (
	"strings"
	"testing"
)

func TestValidateRejectsProductionDefaultS3Credentials(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Environment: "production",
		DatabaseURL: "postgres://db",
		WorkerToken: "production-worker-token",
		InboundS3: S3Config{
			Endpoint:  "s3.example.com",
			Bucket:    "order-fill-inbound",
			AccessKey: "minioadmin",
			SecretKey: "secret",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected default credentials error")
	}
}

func TestValidateRejectsProductionMissingS3(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s3   S3Config
	}{
		{name: "empty endpoint", s3: S3Config{Bucket: "b", AccessKey: "access", SecretKey: "secret"}},
		{name: "empty bucket", s3: S3Config{Endpoint: "s3.example.com", AccessKey: "access", SecretKey: "secret"}},
		{name: "default secret", s3: S3Config{Endpoint: "s3.example.com", Bucket: "b", AccessKey: "access", SecretKey: "minioadmin"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				Environment: "production",
				DatabaseURL: "postgres://db",
				WorkerToken: "production-worker-token",
				InboundS3:   tc.s3,
			}
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected production S3 validation error")
			}
		})
	}
}

func TestValidateAllowsLocalDefaultS3(t *testing.T) {
	t.Parallel()
	cfg := Config{Environment: "local", WorkerToken: "local-dev-worker-token"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateProductionOK(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Environment: "production",
		DatabaseURL: "postgres://db",
		WorkerToken: "production-worker-token",
		InboundS3: S3Config{
			Endpoint:  "s3.example.com",
			Bucket:    "order-fill-inbound",
			AccessKey: "access",
			SecretKey: "secret",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}

func TestValidateErrorMentionsDefaultCredentials(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Environment: "production",
		DatabaseURL: "postgres://db",
		WorkerToken: "production-worker-token",
		InboundS3: S3Config{
			Endpoint: "s3.example.com", Bucket: "b", AccessKey: "minioadmin", SecretKey: "minioadmin",
		},
	}
	err := cfg.Validate()
	if err == nil || (!strings.Contains(err.Error(), "minioadmin") && !strings.Contains(err.Error(), "default")) {
		t.Fatalf("got %v", err)
	}
}
