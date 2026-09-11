package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
