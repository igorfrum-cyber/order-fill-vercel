package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("create inbound object store client: %w", err)
	}
	return &MinIO{client: client, bucket: bucket}, nil
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
