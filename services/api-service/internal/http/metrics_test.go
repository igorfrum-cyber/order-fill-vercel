package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsEndpointReturnsSnapshot(t *testing.T) {
	router := NewRouter(WithMetrics(staticMetrics{"jobs_created": 2}))
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if got := strings.TrimSpace(response.Body.String()); got != `{"jobs_created":2}` {
		t.Fatalf("unexpected metrics body %q", got)
	}
}

type staticMetrics map[string]int64

func (m staticMetrics) Snapshot() map[string]int64 {
	return m
}
