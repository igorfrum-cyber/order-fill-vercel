package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
)

func gzipJSONForTest(t *testing.T, value any) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func previewJob() job.Job {
	return job.Job{
		ID:     "job-1",
		Type:   job.TypeOrderFill,
		Status: job.StatusNeedsReview,
		OutputFiles: []job.OutputFile{
			{ID: "output-2", Name: "Заказ заполненный.xlsx", StorageKey: "jobs/job-1/outputs/source.xlsx"},
		},
	}
}

func previewMetaPayload() map[string]any {
	return map[string]any{
		"chunk_rows": 2,
		"sheets": []map[string]any{
			{
				"name":           "Тюмень",
				"index":          0,
				"max_row":        5,
				"max_column":     4,
				"header_row":     1,
				"article_column": 1,
				"articles":       map[string]int{"RG01": 4, "CT02": 5},
			},
		},
	}
}

func seedPreview(t *testing.T, storage *fakeStorage) {
	t.Helper()
	storage.objects["jobs/job-1/preview/output-2/meta.json.gz"] = port.Object{
		Content:     gzipJSONForTest(t, previewMetaPayload()),
		ContentType: "application/gzip",
	}
	storage.objects["jobs/job-1/preview/output-2/s0/c0.json.gz"] = port.Object{
		Content: gzipJSONForTest(t, map[string]any{
			"start_row": 1,
			"end_row":   2,
			"rows":      [][]string{{"Артикул", "Товар"}, {"", ""}},
		}),
		ContentType: "application/gzip",
	}
	storage.objects["jobs/job-1/preview/output-2/s0/c1.json.gz"] = port.Object{
		Content: gzipJSONForTest(t, map[string]any{
			"start_row": 3,
			"end_row":   4,
			"rows":      [][]string{{"x"}, {"RG01", "Крем", "12", "коробка"}},
		}),
		ContentType: "application/gzip",
	}
	storage.objects["jobs/job-1/preview/output-2/s0/c2.json.gz"] = port.Object{
		Content: gzipJSONForTest(t, map[string]any{
			"start_row": 5,
			"end_row":   5,
			"rows":      [][]string{{"CT02"}},
		}),
		ContentType: "application/gzip",
	}
}

func TestPreviewMetaDoesNotLoadChunks(t *testing.T) {
	repository := newFakeRepository()
	repository.stored["job-1"] = previewJob()
	storage := newFakeStorage()
	seedPreview(t, storage)
	gets := 0
	counting := countingStore{Store: storage, onGet: func(string) { gets++ }}

	meta, err := NewPreviewReader(repository, counting).Meta(context.Background(), "job-1", "output-2")
	if err != nil {
		t.Fatal(err)
	}
	if gets != 1 {
		t.Fatalf("meta must load one object, got %d gets", gets)
	}
	if len(meta.Sheets) != 1 || meta.Sheets[0].Name != "Тюмень" || meta.Sheets[0].Articles["RG01"] != 4 {
		t.Fatalf("meta %#v", meta)
	}
}

func TestPreviewWindowLoadsOnlyOverlappingChunks(t *testing.T) {
	repository := newFakeRepository()
	repository.stored["job-1"] = previewJob()
	storage := newFakeStorage()
	seedPreview(t, storage)
	var keys []string
	counting := countingStore{Store: storage, onGet: func(key string) { keys = append(keys, key) }}

	window, err := NewPreviewReader(repository, counting).Window(context.Background(), PreviewWindowQuery{
		JobID: "job-1", FileID: "output-2", SheetIndex: 0, FromRow: 4, ToRow: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if window.FromRow != 4 || window.ToRow != 5 {
		t.Fatalf("bounds %#v", window)
	}
	if len(window.Rows) != 2 || window.Rows[0][0] != "RG01" || window.Rows[1][0] != "CT02" {
		t.Fatalf("rows %#v", window.Rows)
	}
	if len(window.Rows[0]) != 4 {
		t.Fatalf("rows must be padded to max_column, got %d", len(window.Rows[0]))
	}
	for _, key := range keys {
		if key == "jobs/job-1/preview/output-2/s0/c0.json.gz" {
			t.Fatalf("window must not load a non-overlapping chunk: %v", keys)
		}
	}
}

func TestPreviewWindowRejectsAHugeRange(t *testing.T) {
	repository := newFakeRepository()
	repository.stored["job-1"] = previewJob()
	storage := newFakeStorage()
	seedPreview(t, storage)

	_, err := NewPreviewReader(repository, storage).Window(context.Background(), PreviewWindowQuery{
		JobID: "job-1", FileID: "output-2", FromRow: 1, ToRow: 500,
	})
	if !errors.Is(err, job.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestPreviewFindJumpsToTheArticleRow(t *testing.T) {
	repository := newFakeRepository()
	repository.stored["job-1"] = previewJob()
	storage := newFakeStorage()
	seedPreview(t, storage)

	hit, err := NewPreviewReader(repository, storage).Find(context.Background(), "job-1", "output-2", 0, "rg01")
	if err != nil {
		t.Fatal(err)
	}
	if !hit.Found || hit.Row != 4 || hit.Column != 1 {
		t.Fatalf("hit %#v", hit)
	}

	miss, err := NewPreviewReader(repository, storage).Find(context.Background(), "job-1", "output-2", 0, "nope")
	if err != nil {
		t.Fatal(err)
	}
	if miss.Found {
		t.Fatalf("missing article %#v", miss)
	}
}

func TestPreviewMetaNotFoundForUnknownFile(t *testing.T) {
	repository := newFakeRepository()
	repository.stored["job-1"] = previewJob()
	_, err := NewPreviewReader(repository, newFakeStorage()).Meta(context.Background(), "job-1", "output-9")
	if !errors.Is(err, job.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

type countingStore struct {
	Store port.ObjectStore
	onGet func(string)
}

func (s countingStore) Put(ctx context.Context, key string, contentType string, content []byte) error {
	return s.Store.Put(ctx, key, contentType, content)
}

func (s countingStore) Get(ctx context.Context, key string) (port.Object, error) {
	if s.onGet != nil {
		s.onGet(key)
	}
	return s.Store.Get(ctx, key)
}
