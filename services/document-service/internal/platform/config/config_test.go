package config

import "testing"

func TestLoadDefaultsWorkerConcurrencyToOne(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("QUEUE_URL", "redis://redis:6379/0")
	t.Setenv("S3_ENDPOINT", "http://minio:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.WorkerConcurrency != 1 {
		t.Errorf("Load().WorkerConcurrency = %d, want %d", got.WorkerConcurrency, 1)
	}
}

func TestLoadReadsWorkerConcurrency(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("QUEUE_URL", "redis://redis:6379/0")
	t.Setenv("S3_ENDPOINT", "http://minio:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")
	t.Setenv("WORKER_CONCURRENCY", "4")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.WorkerConcurrency != 4 {
		t.Errorf("Load().WorkerConcurrency = %d, want %d", got.WorkerConcurrency, 4)
	}
}

func TestLoadRejectsInvalidWorkerConcurrency(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("QUEUE_URL", "redis://redis:6379/0")
	t.Setenv("S3_ENDPOINT", "http://minio:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")
	t.Setenv("WORKER_CONCURRENCY", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid WORKER_CONCURRENCY error")
	}
}
