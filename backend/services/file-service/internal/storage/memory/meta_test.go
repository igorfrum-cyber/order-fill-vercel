package memory

import (
	"errors"
	"testing"

	"order-fill/backend/services/file-service/internal/domain"
)

func TestMetaStoreObjectsByIDAndKey(t *testing.T) {
	t.Parallel()

	store := NewMeta()
	if _, err := store.GetByID(t.Context(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID(missing) = %v", err)
	}
	if _, err := store.GetByKey(t.Context(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByKey(missing) = %v", err)
	}

	object := domain.Object{ID: "object-1", Key: "company-1/file.xlsx", Name: "file.xlsx", CompanyID: "company-1", Body: []byte("data")}
	if err := store.SaveObject(t.Context(), object); err != nil {
		t.Fatal(err)
	}
	byID, err := store.GetByID(t.Context(), object.ID)
	if err != nil || byID.ID != object.ID || string(byID.Body) != "data" {
		t.Fatalf("GetByID() = %+v, %v", byID, err)
	}
	byKey, err := store.GetByKey(t.Context(), object.Key)
	if err != nil || byKey.ID != object.ID {
		t.Fatalf("GetByKey() = %+v, %v", byKey, err)
	}

	replacement := domain.Object{ID: "object-2", Key: object.Key, Name: "replacement.xlsx"}
	if err := store.SaveObject(t.Context(), replacement); err != nil {
		t.Fatal(err)
	}
	byKey, err = store.GetByKey(t.Context(), object.Key)
	if err != nil || byKey.ID != replacement.ID {
		t.Fatalf("GetByKey after replace = %+v, %v", byKey, err)
	}
}

func TestMetaStoreUploads(t *testing.T) {
	t.Parallel()

	store := NewMeta()
	if _, err := store.GetUpload(t.Context(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetUpload(missing) = %v", err)
	}
	upload := domain.Upload{ID: "upload-1", Name: "input.xlsx", ContentType: "application/xlsx"}
	if err := store.SaveUpload(t.Context(), upload); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetUpload(t.Context(), upload.ID)
	if err != nil || got != upload {
		t.Fatalf("GetUpload() = %+v, %v", got, err)
	}
}
