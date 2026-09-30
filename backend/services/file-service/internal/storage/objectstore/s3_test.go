package objectstore

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"order-fill/backend/services/file-service/internal/domain"
)

func TestMemoryObjectStorePutAndGetCopiesBytes(t *testing.T) {
	t.Parallel()

	store := NewS3()
	original := []byte("workbook")
	if err := store.Put(t.Context(), "company/file.xlsx", bytes.NewReader(original), int64(len(original)), "application/xlsx"); err != nil {
		t.Fatal(err)
	}
	original[0] = 'X'
	reader, contentType, err := store.Get(t.Context(), "company/file.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil || string(body) != "workbook" || contentType != "application/xlsx" {
		t.Fatalf("Get() = %q, %q, %v", body, contentType, err)
	}
}

func TestMemoryObjectStoreReturnsNotFound(t *testing.T) {
	t.Parallel()

	if _, _, err := NewS3().Get(t.Context(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get(missing) error = %v", err)
	}
}

func TestMemoryObjectStoreSupportsOverwrite(t *testing.T) {
	t.Parallel()

	store := NewS3()
	for _, value := range []string{"first", "second"} {
		if err := store.Put(t.Context(), "key", bytes.NewReader([]byte(value)), -1, value); err != nil {
			t.Fatal(err)
		}
	}
	reader, contentType, err := store.Get(t.Context(), "key")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil || string(body) != "second" || contentType != "second" {
		t.Fatalf("overwritten object = %q %q %v", body, contentType, err)
	}
}
