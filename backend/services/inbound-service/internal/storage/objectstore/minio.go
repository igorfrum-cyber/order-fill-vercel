package objectstore

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"order-fill/backend/services/inbound-service/internal/domain"
)

type MinIO struct {
	client *minio.Client
	bucket string
}

func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIO, error) {
	host, secure, err := parseEndpoint(endpoint, useSSL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("inbound bucket is required")
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
		return nil, fmt.Errorf("create inbound object store client: %w", err)
	}
	return &MinIO{client: client, bucket: bucket}, nil
}

func s3Transport() (*http.Transport, error) {
	caFile := strings.TrimSpace(os.Getenv("GRPC_TLS_CA_FILE"))
	if caFile == "" {
		return nil, nil
	}
	// #nosec G304 -- TLS material paths are operator-owned startup configuration, not request input.
	raw, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read inbound object store CA: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("inbound object store CA does not contain PEM certificates")
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("http default transport is not *http.Transport")
	}
	transport := base.Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}
	return transport, nil
}

func (s *MinIO) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check inbound bucket %s: %w", s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create inbound bucket %s: %w", s.bucket, err)
	}
	return nil
}

func (s *MinIO) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put inbound object %s: %w", key, err)
	}
	return nil
}

func (s *MinIO) Get(ctx context.Context, key string) ([]byte, string, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", wrapGetError(key, err)
	}
	defer func() { _ = object.Close() }()
	info, err := object.Stat()
	if err != nil {
		return nil, "", wrapGetError(key, err)
	}
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, "", fmt.Errorf("read inbound object %s: %w", key, err)
	}
	return data, info.ContentType, nil
}

func wrapGetError(key string, err error) error {
	response := minio.ToErrorResponse(err)
	if response.StatusCode == http.StatusNotFound || response.Code == "NoSuchKey" || response.Code == "NoSuchBucket" || response.Code == "NotFound" {
		return fmt.Errorf("%w: inbound object %s", domain.ErrNotFound, key)
	}
	return fmt.Errorf("get inbound object %s: %w", key, err)
}

func parseEndpoint(endpoint string, useSSL bool) (string, bool, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "", false, fmt.Errorf("inbound object store endpoint is required")
	}
	if !strings.Contains(trimmed, "://") {
		return strings.TrimSuffix(trimmed, "/"), useSSL, nil
	}
	host := strings.TrimSuffix(trimmed, "/")
	if i := strings.Index(host, "://"); i >= 0 {
		scheme := host[:i]
		host = host[i+3:]
		switch scheme {
		case "http":
			return host, false, nil
		case "https":
			return host, true, nil
		default:
			return "", false, fmt.Errorf("unsupported object store scheme %q", scheme)
		}
	}
	return host, useSSL, nil
}
