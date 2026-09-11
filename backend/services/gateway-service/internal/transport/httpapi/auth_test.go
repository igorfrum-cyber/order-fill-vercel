package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPresentUserIncludesDisabledAndLastSeen(t *testing.T) {
	t.Parallel()
	got := presentUser(User{
		ID: "u1", Login: "buyer", Role: "purchaser", CompanyID: "co",
		DisabledAt: "2026-09-01T12:00:00Z", LastSeenAt: "2026-09-11T08:00:00Z",
	})
	if got["disabled_at"] != "2026-09-01T12:00:00Z" || got["last_seen_at"] != "2026-09-11T08:00:00Z" {
		t.Fatalf("got %#v", got)
	}
	active := presentUser(User{ID: "u2", Login: "keeper", Role: "company_admin"})
	if _, ok := active["disabled_at"]; ok {
		t.Fatal("active user must omit disabled_at")
	}
}

func TestTotpDisableRequiresPasswordAndCode(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/2fa/disable", strings.NewReader(`{"password":"secret"}`))
	req = req.WithContext(withUser(req.Context(), User{ID: "u1", Login: "buyer"}))
	rec := httptest.NewRecorder()
	api.totpDisable(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}
