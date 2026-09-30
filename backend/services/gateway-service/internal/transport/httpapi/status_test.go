package httpapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHealthProbe(t *testing.T) {
	t.Parallel()

	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer bad.Close()

	api := &API{}
	if !api.httpOK(t.Context(), good.URL) {
		t.Fatal("healthy endpoint was marked unavailable")
	}
	if api.httpOK(t.Context(), bad.URL) {
		t.Fatal("unhealthy endpoint was marked available")
	}
	if api.httpOK(t.Context(), "  ") || api.httpOK(t.Context(), "://bad-url") {
		t.Fatal("invalid health URLs should fail")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if api.httpOK(ctx, good.URL) {
		t.Fatal("cancelled health request should fail")
	}
}

func TestTCPHealthProbe(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	defer func() { _ = listener.Close() }()

	api := &API{}
	if !api.tcpOK(t.Context(), address) {
		t.Fatal("listening endpoint was marked unavailable")
	}
	if api.tcpOK(t.Context(), "") || api.tcpOK(t.Context(), "bad-address") {
		t.Fatal("empty and invalid endpoints should fail")
	}
}

func TestListStatusRequiresPlatformAdminAndReportsComponents(t *testing.T) {
	t.Parallel()

	unauthorized := httptest.NewRecorder()
	(&API{}).listStatus(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil).WithContext(withUser(t.Context(), User{Role: "company_admin"})))
	if unauthorized.Code != http.StatusNotFound {
		t.Fatalf("non-admin status code = %d", unauthorized.Code)
	}

	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer health.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	defer func() { _ = listener.Close() }()

	api := &API{WorkerHealth: health.URL, FileHealth: health.URL, PostgresAddr: address, RedisAddr: "", InboundGRPC: address}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil).WithContext(withUser(t.Context(), User{ID: "root", Role: "platform_admin"}))
	rec := httptest.NewRecorder()
	api.listStatus(rec, req)
	var payload struct {
		Components []struct {
			ID string `json:"id"`
			OK bool   `json:"ok"`
		} `json:"components"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || len(payload.Components) != 6 {
		t.Fatalf("status=%d payload=%+v", rec.Code, payload)
	}
	for _, component := range payload.Components {
		if component.ID == "redis" && component.OK {
			t.Fatal("empty Redis address should be unhealthy")
		}
	}
}
