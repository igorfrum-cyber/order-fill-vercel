package files_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"

	"order-fill/backend/services/file-service/internal/domain"
	"order-fill/backend/services/file-service/internal/service/files"
	"order-fill/backend/services/file-service/internal/storage/memory"
	"order-fill/backend/services/file-service/internal/storage/objectstore"
)

func TestPutGetPreservesContentType(t *testing.T) {
	t.Parallel()
	svc := files.New(objectstore.NewS3(), memory.NewMeta())
	obj, err := svc.Put(t.Context(), "", `C:\uploads\blank.xlsx`, "application/vnd.ms-excel", []byte("xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	if obj.Name != "blank.xlsx" {
		t.Fatalf("name=%s", obj.Name)
	}
	got, err := svc.Get(t.Context(), obj.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ContentType != "application/vnd.ms-excel" || string(got.Body) != "xlsx" {
		t.Fatalf("got %+v", got)
	}
}

type errMeta struct {
	files.MetaStore
	saveErr error
}

func (m errMeta) SaveObject(domain.Object) error { return m.saveErr }

func TestPutSurfacesMetaError(t *testing.T) {
	t.Parallel()
	svc := files.New(objectstore.NewS3(), errMeta{MetaStore: memory.NewMeta(), saveErr: errors.New("meta down")})
	_, err := svc.Put(t.Context(), "", "a.xlsx", "text/plain", []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "meta down") {
		t.Fatalf("got %v", err)
	}
}

func TestMissingObject(t *testing.T) {
	t.Parallel()
	svc := files.New(objectstore.NewS3(), memory.NewMeta())
	if _, err := svc.Get(t.Context(), "missing", ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestArchiveAndIdempotentFinalize(t *testing.T) {
	t.Parallel()
	svc := files.New(objectstore.NewS3(), memory.NewMeta())
	a, err := svc.Put(t.Context(), "", "a.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", []byte("A"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Put(t.Context(), "", "b.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", []byte("B"))
	if err != nil {
		t.Fatal(err)
	}
	zipObj, err := svc.Archive(t.Context(), []string{a.ID, b.ID}, "pack.zip")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(t.Context(), zipObj.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ContentType != "application/zip" {
		t.Fatalf("ctype=%s", got.ContentType)
	}
	zr, err := zip.NewReader(bytes.NewReader(got.Body), int64(len(got.Body)))
	if err != nil || len(zr.File) != 2 {
		t.Fatalf("zip files=%v err=%v", len(zr.File), err)
	}

	up, err := svc.CreateUpload(t.Context(), "c.xlsx", "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.FinalizeUpload(t.Context(), up.ID, []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.FinalizeUpload(t.Context(), up.ID, []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("finalize must be idempotent: %s vs %s", first.ID, second.ID)
	}
	body, err := svc.Get(t.Context(), first.ID, "")
	if err != nil || string(body.Body) != "one" {
		t.Fatalf("body=%q err=%v", body.Body, err)
	}
}
