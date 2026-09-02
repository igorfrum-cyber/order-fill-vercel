package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubMetrics struct {
	snapshot map[string]int64
}

func (s stubMetrics) Snapshot() map[string]int64 {
	return s.snapshot
}

func TestHealthz(t *testing.T) {
	response := do(t, NewRouter(stubMetrics{}), "/healthz")

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content type: got %q", contentType)
	}
	if body := strings.TrimSpace(response.Body.String()); body != `{"status":"ok"}` {
		t.Fatalf("body: got %q", body)
	}
}

func TestMetrics(t *testing.T) {
	metrics := stubMetrics{snapshot: map[string]int64{
		"jobs_completed":         2,
		"jobs_failed":            1,
		"processing_duration_ms": 42,
	}}
	response := do(t, NewRouter(metrics), "/metrics")

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d", response.Code, http.StatusOK)
	}

	var decoded map[string]int64
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	for key, want := range metrics.snapshot {
		if decoded[key] != want {
			t.Fatalf("%s: got %d want %d", key, decoded[key], want)
		}
	}
}

func TestUnknownPath(t *testing.T) {
	if response := do(t, NewRouter(stubMetrics{}), "/nope"); response.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want %d", response.Code, http.StatusNotFound)
	}
}

func do(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
