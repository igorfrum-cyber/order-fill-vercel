package healthz

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiveOK(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	Live().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("status=%q", body.Status)
	}
}

func TestReady(t *testing.T) {
	t.Parallel()
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		Ready(func(context.Context) error { return nil }).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
	})
	t.Run("not ready", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		Ready(func(context.Context) error { return errors.New("postgres down") }).ServeHTTP(
			rec, httptest.NewRequest(http.MethodGet, "/readyz", nil),
		)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d", rec.Code)
		}
		var body Response
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Status != "not_ready" || body.Error != "postgres down" {
			t.Fatalf("body=%+v", body)
		}
	})
}
