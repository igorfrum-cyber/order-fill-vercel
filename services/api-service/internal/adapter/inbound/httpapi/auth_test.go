package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
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

func TestSecurityHeadersOnResponses(t *testing.T) {
	router := NewRouter(Config{AllowedOrigins: []string{"http://127.0.0.1:3200"}})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	header := response.Header()
	if got := header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options: got %q", got)
	}
	if got := header.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options: got %q", got)
	}
	if got := header.Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy: got %q", got)
	}
	if header.Get("Permissions-Policy") == "" {
		t.Fatal("Permissions-Policy must be present")
	}
	if header.Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy must be present")
	}
}

func TestMeIncludesCompanyName(t *testing.T) {
	token := "owner-token"
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Auth: stubAuth{users: map[string]identity.User{
			identity.HashSecret(token): {
				ID:          "user-a",
				Login:       "buyer",
				Role:        identity.RolePurchaser,
				CompanyID:   "company-a",
				CompanyName: "Acme",
			},
		}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body %s", response.Code, response.Body.String())
	}
	var payload map[string]string
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["company_name"] != "Acme" {
		t.Fatalf("company_name: got %#v", payload["company_name"])
	}
}

func TestSubmitEditsRejectsOversizedBody(t *testing.T) {
	owner := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser, Login: "a"}
	entity := job.Job{ID: "job-1", CompanyID: "company-a", CreatedBy: owner.ID, Status: job.StatusNeedsReview}
	token := "owner-token"
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Auth: stubAuth{users: map[string]identity.User{
			identity.HashSecret(token): owner,
		}},
		GetJob:      stubJobFinder{jobs: map[string]job.Job{entity.ID: entity}},
		SubmitEdits: stubEditor{},
	})
	body := `{"edits":[{"key":"k","value":"` + strings.Repeat("x", authJSONLimit) + `"}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/job-1/edits", strings.NewReader(body))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body %s", response.Code, response.Body.String())
	}
}

type stubEditor struct{}

func (stubEditor) Execute(_ context.Context, _ string, _ []job.ManualEdit) (job.Job, error) {
	return job.Job{ID: "job-1", Status: job.StatusCompleted}, nil
}
