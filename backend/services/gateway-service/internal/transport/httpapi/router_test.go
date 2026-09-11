package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"order-fill/backend/services/gateway-service/internal/clients"
	"order-fill/backend/services/gateway-service/internal/config"
)

func TestHealthAndCSRF(t *testing.T) {
	t.Parallel()
	h := New(config.Config{AllowedOrigins: "*"}, clients.Clients{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health status=%d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("cache-control=%q", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" {
		t.Fatalf("coop=%q", rec.Header().Get("Cross-Origin-Opener-Policy"))
	}
	if rec.Header().Get("Cross-Origin-Resource-Policy") != "same-origin" {
		t.Fatalf("corp=%q", rec.Header().Get("Cross-Origin-Resource-Policy"))
	}
	post := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/order-fill", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, post)
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
		t.Fatalf("csrf/auth status=%d", rec.Code)
	}
}

func TestCSRFAllowsCompanyLocalhostSubdomain(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	req.Header.Set("X-Requested-With", "fetch")
	req.Header.Set("Origin", "http://kristail.localhost:3200")
	if !csrfAllowed(req, []string{"http://127.0.0.1:3200"}) {
		t.Fatal("company localhost subdomain must pass")
	}
}

func TestCSRFAllowsCompanySubdomainOfAllowedOrigin(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	req.Header.Set("X-Requested-With", "fetch")
	req.Header.Set("Origin", "https://kristail.example.com")
	if !csrfAllowed(req, []string{"https://example.com"}) {
		t.Fatal("company subdomain of the allowed origin must pass")
	}
}

func TestCSRFRejectsUnrelatedSubdomain(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	req.Header.Set("X-Requested-With", "fetch")
	req.Header.Set("Origin", "https://evil.example.net")
	if csrfAllowed(req, []string{"https://example.com"}) {
		t.Fatal("foreign host must not pass")
	}
}

func TestPublicLoginRejectsForeignOrigin(t *testing.T) {
	t.Parallel()
	h := New(config.Config{AllowedOrigins: "http://127.0.0.1:3200"}, clients.Clients{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestPublicLoginRejectsMissingOrigin(t *testing.T) {
	t.Parallel()
	h := New(config.Config{AllowedOrigins: "http://127.0.0.1:3200"}, clients.Clients{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
}
