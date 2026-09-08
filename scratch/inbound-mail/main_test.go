package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, want string
	}{
		{"plain", "report.xlsx", "report.xlsx"},
		{"slash", "../../etc/passwd", "passwd"},
		{"empty", "", "unnamed"},
		{"dots", "..", "unnamed"},
		{"spaces", "a b.xlsx", "a_b.xlsx"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := safeName(tc.in); got != tc.want {
				t.Fatalf("safeName(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTokenOK(t *testing.T) {
	t.Parallel()
	if tokenOK("abc", "abc") != true {
		t.Fatal("equal tokens must pass")
	}
	if tokenOK("abd", "abc") {
		t.Fatal("different tokens must fail")
	}
	if tokenOK("ab", "abc") {
		t.Fatal("length mismatch must fail")
	}
}

func TestDumpJSONAttachment(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw := `{"attachments":[{"file_name":"a.xlsx","content":"` + base64.StdEncoding.EncodeToString([]byte("xlsx")) + `"}]}`
	req := httptest.NewRequest(http.MethodPost, "/inbound", bytes.NewReader([]byte(raw)))
	req.Header.Set("Content-Type", "application/json")
	saved, err := dump(dir, req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(saved, "01-a.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "xlsx" {
		t.Fatalf("got %q", got)
	}
}

func TestDumpJSONAttachmentURL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw := `{"attachments":[{"file_name":"a.xlsx","url":"https://example.com/a.xlsx"}]}`
	req := httptest.NewRequest(http.MethodPost, "/inbound", strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	saved, err := dump(dir, req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(saved, "01-a.xlsx.url.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != "https://example.com/a.xlsx" {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"envelope":{"from":"a@b.test","to":"in@c.test"},
		"headers":{"subject":"продажи 8 сен","from":"A <a@b.test>"},
		"attachments":[{"file_name":"sales.xlsx"}]
	}`)
	got := summarizeJSON(raw)
	if got.From != "a@b.test" || got.To != "in@c.test" || got.Subject != "продажи 8 сен" {
		t.Fatalf("got %+v", got)
	}
	if len(got.Files) != 1 || got.Files[0] != "sales.xlsx" {
		t.Fatalf("files %v", got.Files)
	}
}

func TestDumpWritesMeta(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw := `{"envelope":{"from":"a@b.test"},"headers":{"subject":"hi"},"attachments":[]}`
	req := httptest.NewRequest(http.MethodPost, "/inbound", strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	saved, err := dump(dir, req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(saved, "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"subject": "hi"`) {
		t.Fatalf("meta %s", got)
	}
}

func TestInboxPage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw := `{"envelope":{"from":"a@b.test"},"headers":{"subject":"продажи"},"attachments":[]}`
	req := httptest.NewRequest(http.MethodPost, "/inbound", strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if _, err := dump(dir, req); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newMux("secret", dir))
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: %d", res.StatusCode)
	}

	res, err = http.Get(srv.URL + "/?token=secret")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "продажи") || !strings.Contains(string(body), "a@b.test") {
		t.Fatalf("page %s", body)
	}
}
