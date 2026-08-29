package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterAllowsBrowserOriginByDefault(t *testing.T) {
	router := NewRouter(Config{})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "http://localhost:3200")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard allow origin, got %q", got)
	}
}

func TestRouterAnswersPreflight(t *testing.T) {
	router := NewRouter(Config{AllowedOrigins: []string{"http://localhost:3200"}})
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/jobs/order-fill", nil)
	request.Header.Set("Origin", "http://localhost:3200")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3200" {
		t.Fatalf("expected origin echo, got %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != corsAllowedHeaders {
		t.Fatalf("expected allowed headers %q, got %q", corsAllowedHeaders, got)
	}
}

func TestRouterRejectsUnknownOrigin(t *testing.T) {
	router := NewRouter(Config{AllowedOrigins: []string{"http://localhost:3200"}})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "http://evil.example")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow origin header, got %q", got)
	}
}

func TestParseAllowedOriginsFallsBackToWildcard(t *testing.T) {
	if got := ParseAllowedOrigins("  "); len(got) != 1 || got[0] != "*" {
		t.Fatalf("expected wildcard fallback, got %v", got)
	}
	got := ParseAllowedOrigins("http://localhost:3200/, http://127.0.0.1:3200")
	if len(got) != 2 || got[0] != "http://localhost:3200" || got[1] != "http://127.0.0.1:3200" {
		t.Fatalf("unexpected origins %v", got)
	}
}

func TestHealthzReturnsOK(t *testing.T) {
	router := NewRouter(Config{})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if got := response.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected body %q", got)
	}
}
