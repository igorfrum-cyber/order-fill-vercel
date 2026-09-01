package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"order-fill/services/api-service/internal/app/usecase"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func TestPurchaserCannotReadPeerJob(t *testing.T) {
	owner := identity.User{ID: "user-a", CompanyID: "company-a", Role: identity.RolePurchaser, Login: "a"}
	peer := identity.User{ID: "user-b", CompanyID: "company-a", Role: identity.RolePurchaser, Login: "b"}
	entity := job.Job{ID: "job-1", CompanyID: "company-a", CreatedBy: owner.ID, Status: job.StatusCompleted}

	ownerToken := "owner-token"
	peerToken := "peer-token"
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Auth: stubAuth{users: map[string]identity.User{
			identity.HashSecret(ownerToken): owner,
			identity.HashSecret(peerToken):  peer,
		}},
		GetJob: stubJobFinder{jobs: map[string]job.Job{entity.ID: entity}},
	})

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/jobs/job-1"},
		{http.MethodGet, "/api/v1/jobs/job-1/report"},
		{http.MethodPost, "/api/v1/jobs/job-1/edits"},
		{http.MethodGet, "/api/v1/jobs/job-1/files"},
		{http.MethodGet, "/api/v1/jobs/job-1/archive"},
		{http.MethodGet, "/api/v1/jobs/job-1/files/output-1"},
		{http.MethodGet, "/api/v1/jobs/job-1/files/output-1/preview"},
	}
	for _, path := range paths {
		t.Run(path.method+" "+path.path+" peer", func(t *testing.T) {
			response := doAuthed(router, path.method, path.path, peerToken)
			if response.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d body %s", response.Code, response.Body.String())
			}
		})
	}

	owned := doAuthed(router, http.MethodGet, "/api/v1/jobs/job-1", ownerToken)
	if owned.Code != http.StatusOK {
		t.Fatalf("owner should read own job, got %d body %s", owned.Code, owned.Body.String())
	}
}

func TestOtherCompanyAdminCannotReadJob(t *testing.T) {
	admin := identity.User{ID: "admin-b", CompanyID: "company-b", Role: identity.RoleCompanyAdmin, Login: "admin-b"}
	entity := job.Job{ID: "job-1", CompanyID: "company-a", CreatedBy: "user-a", Status: job.StatusCompleted}
	token := "admin-token"
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Auth: stubAuth{users: map[string]identity.User{
			identity.HashSecret(token): admin,
		}},
		GetJob: stubJobFinder{jobs: map[string]job.Job{entity.ID: entity}},
	})
	response := doAuthed(router, http.MethodGet, "/api/v1/jobs/job-1", token)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

type stubAuth struct {
	users map[string]identity.User
}

func (s stubAuth) Login(context.Context, string, string) (usecase.Session, error) {
	return usecase.Session{}, identity.ErrUnauthorized
}

func (s stubAuth) Logout(context.Context, string) error { return nil }

func (s stubAuth) AcceptInvite(context.Context, string, string) (usecase.Session, error) {
	return usecase.Session{}, identity.ErrUnauthorized
}

func (s stubAuth) SessionUser(_ context.Context, tokenHash string) (identity.User, error) {
	user, ok := s.users[tokenHash]
	if !ok {
		return identity.User{}, identity.ErrUnauthorized
	}
	return user, nil
}

type stubJobFinder struct {
	jobs map[string]job.Job
}

func (s stubJobFinder) Execute(_ context.Context, jobID string) (job.Job, error) {
	entity, ok := s.jobs[jobID]
	if !ok {
		return job.Job{}, job.ErrNotFound
	}
	return entity, nil
}

func doAuthed(router http.Handler, method string, path string, rawToken string) *httptest.ResponseRecorder {
	var body *strings.Reader
	if method == http.MethodPost {
		body = strings.NewReader(`{"edits":[]}`)
	} else {
		body = strings.NewReader("")
	}
	request := httptest.NewRequest(method, path, body)
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawToken})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
