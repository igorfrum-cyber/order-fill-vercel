package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJobCreateRequiresAuthentication(t *testing.T) {
	router := NewRouter(Config{AllowedOrigins: []string{"http://127.0.0.1:3200"}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/order-fill", nil)
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestJobCreateRequiresCSRFHeader(t *testing.T) {
	router := NewRouter(Config{AllowedOrigins: []string{"http://127.0.0.1:3200"}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/order-fill", strings.NewReader("{}"))
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestLimiterRejectsBurst(t *testing.T) {
	limiter := NewLimiter(time.Minute, 2)
	fixed := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return fixed }
	if !limiter.Allow("a") {
		t.Fatal("first allow should pass")
	}
	if !limiter.Allow("a") {
		t.Fatal("second allow should pass")
	}
	if limiter.Allow("a") {
		t.Fatal("third should fail")
	}
}

func TestCSRFAllowsLocalhostWhenLoopbackIsListed(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/companies", strings.NewReader(`{"name":"Acme"}`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://localhost:3200")
	if !csrfAllowed(request, []string{"http://127.0.0.1:3200"}) {
		t.Fatal("localhost:3200 should be accepted when 127.0.0.1:3200 is allowed")
	}
}

func TestCSRFRejectsDifferentPortLoopback(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/companies", strings.NewReader(`{"name":"Acme"}`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://localhost:9999")
	if csrfAllowed(request, []string{"http://127.0.0.1:3200"}) {
		t.Fatal("loopback on another port must not pass")
	}
}
