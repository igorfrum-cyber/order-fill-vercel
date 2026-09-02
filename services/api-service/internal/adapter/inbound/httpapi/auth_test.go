package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func TestPublicCompanyLoginReturnsNameAndSlug(t *testing.T) {
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Admin: stubPublicCompany{
			company: identity.Company{Name: "Acme", LoginSlug: "acme"},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/companies/acme/login", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["name"] != "Acme" || payload["login_slug"] != "acme" {
		t.Fatalf("payload %#v", payload)
	}
	if payload["has_logo"] != false {
		t.Fatalf("has_logo: got %#v", payload["has_logo"])
	}
	if _, ok := payload["id"]; ok {
		t.Fatal("public login must not return company id")
	}
}

func TestPublicCompanyLoginUnknownSlugIsNotFound(t *testing.T) {
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Admin:          stubPublicCompany{missing: true},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/companies/missing/login", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body %s", response.Code, response.Body.String())
	}
}

func TestPublicCompanyLogoServesImage(t *testing.T) {
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Admin: stubPublicCompany{
			company: identity.Company{Name: "Acme", LoginSlug: "acme", LogoContentType: "image/png"},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/companies/acme/logo", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content type: got %q", got)
	}
	if response.Body.String() != "png-bytes" {
		t.Fatalf("body: got %q", response.Body.String())
	}
}

func TestPublicCompanyLoginDisabledCompanyIsNotFound(t *testing.T) {
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Admin:          stubPublicCompany{missing: true},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/companies/gone/login", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body %s", response.Code, response.Body.String())
	}
}

type stubPublicCompany struct {
	company identity.Company
	missing bool
}

func (s stubPublicCompany) CreateCompany(context.Context, identity.User, string, string) (identity.Company, error) {
	return identity.Company{}, identity.ErrNotFound
}
func (s stubPublicCompany) ListCompanies(context.Context, identity.User) ([]identity.Company, error) {
	return nil, identity.ErrNotFound
}
func (s stubPublicCompany) SetCompanyLoginSlug(context.Context, identity.User, string, string) (identity.Company, error) {
	return identity.Company{}, identity.ErrNotFound
}
func (s stubPublicCompany) UpdateCompany(context.Context, identity.User, string, string, string) (identity.Company, error) {
	return identity.Company{}, identity.ErrNotFound
}
func (s stubPublicCompany) SetCompanyLogo(context.Context, identity.User, string, []byte) (identity.Company, error) {
	return identity.Company{}, identity.ErrNotFound
}
func (s stubPublicCompany) ClearCompanyLogo(context.Context, identity.User, string) (identity.Company, error) {
	return identity.Company{}, identity.ErrNotFound
}
func (s stubPublicCompany) DisableCompany(context.Context, identity.User, string) error {
	return identity.ErrNotFound
}
func (s stubPublicCompany) CreateUser(context.Context, identity.User, string, string, identity.Role) (identity.User, string, error) {
	return identity.User{}, "", identity.ErrNotFound
}
func (s stubPublicCompany) ListUsers(context.Context, identity.User, string) ([]identity.User, error) {
	return nil, identity.ErrNotFound
}
func (s stubPublicCompany) DisableUser(context.Context, identity.User, string) error {
	return identity.ErrNotFound
}
func (s stubPublicCompany) ListAudit(context.Context, identity.User) ([]port.AuditEvent, error) {
	return nil, identity.ErrNotFound
}
func (s stubPublicCompany) RecordAudit(context.Context, identity.User, string, string, string) {}
func (s stubPublicCompany) PublicCompanyLogin(context.Context, string) (identity.Company, error) {
	if s.missing {
		return identity.Company{}, identity.ErrNotFound
	}
	return s.company, nil
}
func (s stubPublicCompany) PublicCompanyLogo(context.Context, string) (port.Object, error) {
	if s.missing || !s.company.HasLogo() {
		return port.Object{}, identity.ErrNotFound
	}
	return port.Object{Content: []byte("png-bytes"), ContentType: "image/png"}, nil
}

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

func TestCSRFAllowsCompanyLocalhostSubdomain(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"buyer","password":"correct-horse"}`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://kristail.localhost:3200")
	if !csrfAllowed(request, []string{"http://127.0.0.1:3200"}) {
		t.Fatal("kristail.localhost:3200 should be accepted when 127.0.0.1:3200 is allowed")
	}
}

func TestCSRFAllowsCompanySubdomainOfAllowedOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"buyer","password":"correct-horse"}`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "https://kristail.example.com")
	if !csrfAllowed(request, []string{"https://example.com"}) {
		t.Fatal("company subdomain of the allowed origin must pass")
	}
}

func TestCSRFRejectsUnrelatedSubdomain(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"buyer","password":"correct-horse"}`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "https://evil.example.net")
	if csrfAllowed(request, []string{"https://example.com"}) {
		t.Fatal("foreign host must not pass")
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

func TestMeIncludesCompanyLoginSlug(t *testing.T) {
	token := "owner-token"
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Auth: stubAuth{users: map[string]identity.User{
			identity.HashSecret(token): {
				ID:               "user-a",
				Login:            "admin",
				Role:             identity.RoleCompanyAdmin,
				CompanyID:        "company-a",
				CompanyName:      "Acme",
				CompanyLoginSlug: "acme",
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
	if payload["login_slug"] != "acme" {
		t.Fatalf("login_slug: got %#v", payload["login_slug"])
	}
}

func TestLogoutEverywhereClearsCookie(t *testing.T) {
	token := "owner-token"
	user := identity.User{ID: "user-a", Login: "buyer", Role: identity.RolePurchaser, CompanyID: "company-a"}
	var got identity.User
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Auth: stubAuth{
			users: map[string]identity.User{identity.HashSecret(token): user},
			logoutEverywhere: func(actor identity.User) error {
				got = actor
				return nil
			},
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-everywhere", strings.NewReader("{}"))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body %s", response.Code, response.Body.String())
	}
	if got.ID != user.ID {
		t.Fatalf("logout everywhere actor: got %#v", got)
	}
	cleared := false
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected cleared session cookie")
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
