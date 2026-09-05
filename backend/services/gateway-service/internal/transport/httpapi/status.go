package httpapi

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func (a *API) listStatus(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	if user.Role != "platform_admin" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, map[string]any{"components": []map[string]any{
		{"id": "api", "ok": true},
		{"id": "worker", "ok": a.httpOK(ctx, a.WorkerHealth)},
		{"id": "postgres", "ok": a.tcpOK(ctx, a.PostgresAddr)},
		{"id": "queue", "ok": a.tcpOK(ctx, a.RedisAddr)},
		{"id": "files", "ok": a.httpOK(ctx, a.FileHealth)},
	}})
}

func (a *API) httpOK(ctx context.Context, rawURL string) bool {
	if strings.TrimSpace(rawURL) == "" {
		return false
	}
	client := a.HTTP
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
	return resp.StatusCode < 400
}

func (a *API) tcpOK(ctx context.Context, addr string) bool {
	if strings.TrimSpace(addr) == "" {
		return false
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	return conn.Close() == nil
}
