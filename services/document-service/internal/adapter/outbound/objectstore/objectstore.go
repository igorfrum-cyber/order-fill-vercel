// Package objectstore reads uploaded workbooks and stores generated ones in S3
// compatible storage.
package objectstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"order-fill/services/document-service/internal/app/port"
)

// DefaultContentType is used when the caller did not provide one; every file
// this service stores is an Office Open XML workbook.
const DefaultContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

var _ port.ObjectStore = (*Store)(nil)

// Store is the MinIO backed object store.
type Store struct {
	client *minio.Client
	bucket string
}

// NewStore connects to S3 compatible storage. The endpoint may carry a scheme
// (http://minio:9000); minio-go wants a bare host:port and derives TLS from the
// scheme instead.
func NewStore(endpoint string, accessKey string, secretKey string, bucket string) (*Store, error) {
	host, useSSL, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("object store bucket is required")
	}

	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create object store client: %w", err)
	}
	return &Store{client: client, bucket: bucket}, nil
}

// EnsureBucket creates the bucket when it is missing.
func (s *Store) EnsureBucket(ctx context.Context) error {
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

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	defer func() { _ = object.Close() }()

	content, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", key, err)
	}
	return content, nil
}

func (s *Store) Put(ctx context.Context, key string, contentType string, content []byte) error {
	if strings.TrimSpace(contentType) == "" {
		contentType = DefaultContentType
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

// parseEndpoint splits a configured endpoint into the bare host:port minio-go
// expects and the TLS flag derived from the scheme.
func parseEndpoint(endpoint string) (string, bool, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "", false, fmt.Errorf("object store endpoint is required")
	}
	if !strings.Contains(trimmed, "://") {
		return strings.TrimSuffix(trimmed, "/"), false, nil
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
