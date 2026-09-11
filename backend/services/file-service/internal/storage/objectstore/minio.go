package objectstore

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"order-fill/backend/services/file-service/internal/domain"
)

type MinIO struct {
	client *minio.Client
	bucket string
}

func NewMinIO(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIO, error) {
	host, secure, err := parseEndpoint(endpoint, useSSL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("object store bucket is required")
	}
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	}
	if secure {
		transport, err := s3Transport()
		if err != nil {
			return nil, err
		}
		if transport != nil {
			opts.Transport = transport
		}
	}
	client, err := minio.New(host, opts)
	if err != nil {
		return nil, fmt.Errorf("create object store client: %w", err)
	}
	return &MinIO{client: client, bucket: bucket}, nil
}

func (s *MinIO) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %s: %w", s.bucket, err)
	}
	return nil
}

func (s *MinIO) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (s *MinIO) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", wrapGetError(key, err)
	}
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, "", wrapGetError(key, err)
	}
	return object, info.ContentType, nil
}

func wrapGetError(key string, err error) error {
	response := minio.ToErrorResponse(err)
	if response.StatusCode == http.StatusNotFound || response.Code == "NoSuchKey" || response.Code == "NoSuchBucket" || response.Code == "NotFound" {
		return fmt.Errorf("%w: object %s", domain.ErrNotFound, key)
	}
	return fmt.Errorf("get object %s: %w", key, err)
}

func s3Transport() (*http.Transport, error) {
	caFile := strings.TrimSpace(os.Getenv("GRPC_TLS_CA_FILE"))
	if caFile == "" {
		return nil, nil
	}
	// #nosec G304 -- TLS material paths are operator-owned startup configuration, not request input.
	raw, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read object store CA: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("object store CA does not contain PEM certificates")
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("http default transport is not *http.Transport")
	}
	transport := base.Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}
	return transport, nil
}

func parseEndpoint(endpoint string, useSSL bool) (string, bool, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "", false, fmt.Errorf("object store endpoint is required")
	}
	if !strings.Contains(trimmed, "://") {
		return strings.TrimSuffix(trimmed, "/"), useSSL, nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false, fmt.Errorf("parse object store endpoint %q: %w", endpoint, err)
	}
	if parsed.Host == "" {
		return "", false, fmt.Errorf("object store endpoint %q has no host", endpoint)
	}
	switch parsed.Scheme {
	case "http":
		return parsed.Host, false, nil
	case "https":
		return parsed.Host, true, nil
	default:
		return "", false, fmt.Errorf("unsupported object store scheme %q", parsed.Scheme)
	}
}
