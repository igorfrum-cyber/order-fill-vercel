// CloudMailin dump with a tiny inbox page. Not part of the product.
//
//	go run .
//	docker compose up --build
package main

import (
	"cmp"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
)

const maxBody = 32 << 20

func main() {
	token := cmp.Or(os.Getenv("INBOUND_TOKEN"), mustToken())
	dir := cmp.Or(os.Getenv("INBOUND_DIR"), "inbox")
	addr := cmp.Or(os.Getenv("INBOUND_ADDR"), "127.0.0.1:8787")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Fatal(err)
	}

	log.Printf("inbox GET  http://%s/?token=%s", addr, token)
	log.Printf("hook  POST http://%s/inbound?token=%s", addr, token)
	s := &http.Server{
		Addr:              addr,
		Handler:           newMux(token, dir),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	log.Fatal(s.ListenAndServe())
}

func newMux(token, dir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /inbound", gate(token, func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		saved, err := dump(dir, r)
		if err != nil {
			log.Println("dump:", err)
			http.Error(w, "dump failed", http.StatusInternalServerError)
			return
		}
		log.Println("saved", saved)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	}))
	mux.HandleFunc("GET /{$}", gate(token, func(w http.ResponseWriter, r *http.Request) {
		msgs, err := listMessages(dir)
		if err != nil {
			http.Error(w, "list failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := inboxPage{Token: r.URL.Query().Get("token"), Messages: msgs}
		if err := inboxTmpl.Execute(w, page); err != nil {
			log.Println("tmpl:", err)
		}
	}))
	mux.HandleFunc("GET /m/{id}/{name}", gate(token, func(w http.ResponseWriter, r *http.Request) {
		id := safeName(r.PathValue("id"))
		name := safeName(r.PathValue("name"))
		if id == "unnamed" || name == "unnamed" {
			http.NotFound(w, r)
			return
		}
		box, err := os.OpenRoot(filepath.Join(dir, id))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer box.Close()
		http.ServeFileFS(w, r, box.FS(), name)
	}))
	return mux
}

func gate(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !tokenOK(providedToken(r), token) {
			w.Header().Set("WWW-Authenticate", `Basic realm="inbound"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func providedToken(r *http.Request) string {
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	if a := r.Header.Get("Authorization"); a != "" {
		kind, rest, ok := strings.Cut(a, " ")
		if ok && strings.EqualFold(kind, "Bearer") {
			return strings.TrimSpace(rest)
		}
	}
	if _, pass, ok := r.BasicAuth(); ok {
		return pass
	}
	return ""
}

func tokenOK(got, want string) bool {
	if want == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func mustToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b[:])
}

type mailSummary struct {
	ID       string   `json:"id,omitempty"`
	Received string   `json:"received,omitempty"`
	From     string   `json:"from,omitempty"`
	To       string   `json:"to,omitempty"`
	Subject  string   `json:"subject,omitempty"`
	Files    []string `json:"files,omitempty"`
}

func summarizeJSON(body []byte) mailSummary {
	var raw struct {
		Envelope struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"envelope"`
		Headers     json.RawMessage `json:"headers"`
		Attachments []struct {
			FileName string `json:"file_name"`
		} `json:"attachments"`
	}
	if json.Unmarshal(body, &raw) != nil {
		return mailSummary{}
	}
	sum := mailSummary{From: raw.Envelope.From, To: raw.Envelope.To}
	var headers map[string]any
	if json.Unmarshal(raw.Headers, &headers) == nil {
		sum.Subject = headerString(headers, "subject")
		if sum.From == "" {
			sum.From = headerString(headers, "from")
		}
		if sum.To == "" {
			sum.To = headerString(headers, "to")
		}
	}
	for _, a := range raw.Attachments {
		if n := strings.TrimSpace(a.FileName); n != "" {
			sum.Files = append(sum.Files, n)
		}
	}
	return sum
}

func headerString(headers map[string]any, key string) string {
	for k, v := range headers {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch t := v.(type) {
		case string:
			return t
		case []any:
			if len(t) > 0 {
				if s, ok := t[0].(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

func dump(root string, r *http.Request) (string, error) {
	stamp := time.Now().UTC().Format("20060102T150405.000Z0700")
	dir := filepath.Join(root, stamp)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	box, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer box.Close()

	meta := fmt.Sprintf("%s %s\nContent-Type: %s\n", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
	if err := writeExclusive(box, "headers.txt", []byte(meta)); err != nil {
		return dir, err
	}

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/") {
		if err := r.ParseMultipartForm(maxBody); err != nil {
			return dir, err
		}
		return dir, dumpMultipart(box, r)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return dir, err
	}
	if err := writeExclusive(box, "body", body); err != nil {
		return dir, err
	}
	saved, err := dumpJSONAttachments(box, body)
	if err != nil {
		return dir, err
	}
	sum := summarizeJSON(body)
	sum.ID = stamp
	sum.Received = time.Now().UTC().Format(time.RFC3339)
	if len(saved) > 0 {
		sum.Files = saved
	}
	raw, err := json.MarshalIndent(sum, "", "  ")
	if err != nil {
		return dir, err
	}
	return dir, writeExclusive(box, "meta.json", raw)
}

func dumpMultipart(box *os.Root, r *http.Request) error {
	form := r.MultipartForm
	if form == nil {
		return nil
	}
	values, err := json.MarshalIndent(form.Value, "", "  ")
	if err != nil {
		return err
	}
	if err := writeExclusive(box, "form.json", values); err != nil {
		return err
	}
	n := 0
	for _, fhs := range form.File {
		for _, fh := range fhs {
			src, err := fh.Open()
			if err != nil {
				return err
			}
			data, err := io.ReadAll(io.LimitReader(src, maxBody+1))
			_ = src.Close()
			if err != nil {
				return err
			}
			if len(data) > maxBody {
				return errors.New("attachment too large")
			}
			n++
			name := fmt.Sprintf("%02d-%s", n, safeName(fh.Filename))
			if err := writeExclusive(box, name, data); err != nil {
				return err
			}
		}
	}
	return nil
}

type inbound struct {
	Attachments []struct {
		FileName string `json:"file_name"`
		Content  string `json:"content"`
		URL      string `json:"url"`
	} `json:"attachments"`
}

func dumpJSONAttachments(box *os.Root, body []byte) ([]string, error) {
	var mail inbound
	if err := json.Unmarshal(body, &mail); err != nil {
		return nil, nil
	}
	var saved []string
	for i, a := range mail.Attachments {
		name := fmt.Sprintf("%02d-%s", i+1, safeName(a.FileName))
		if a.Content != "" {
			data, err := base64.StdEncoding.DecodeString(a.Content)
			if err != nil {
				return saved, err
			}
			if err := writeExclusive(box, name, data); err != nil {
				return saved, err
			}
			saved = append(saved, name)
			continue
		}
		// ponytail: do not GET attachment URLs from the payload (SSRF). save the link instead.
		if a.URL != "" {
			link := name + ".url.txt"
			if err := writeExclusive(box, link, []byte(a.URL+"\n")); err != nil {
				return saved, err
			}
			saved = append(saved, link)
		}
	}
	return saved, nil
}

func listMessages(root string) ([]mailSummary, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]mailSummary, 0, len(ents))
	for _, ent := range ents {
		if !ent.IsDir() {
			continue
		}
		sum := mailSummary{ID: ent.Name()}
		raw, err := os.ReadFile(filepath.Join(root, ent.Name(), "meta.json"))
		if err == nil {
			_ = json.Unmarshal(raw, &sum)
			sum.ID = ent.Name()
		}
		out = append(out, sum)
	}
	slices.SortFunc(out, func(a, b mailSummary) int {
		return strings.Compare(b.ID, a.ID)
	})
	return out, nil
}

func writeExclusive(box *os.Root, name string, p []byte) error {
	f, err := box.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(p)
	return errors.Join(werr, f.Close())
}

func safeName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), ".")
	if out == "" || out == "." || out == ".." {
		return "unnamed"
	}
	return out
}

type inboxPage struct {
	Token    string
	Messages []mailSummary
}

func (p inboxPage) Query() string {
	if p.Token == "" {
		return ""
	}
	return "?token=" + p.Token
}

var inboxTmpl = template.Must(template.New("inbox").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Входящая почта</title>
<style>
body{font-family:sans-serif;max-width:56rem;margin:2rem auto;padding:0 1rem;color:#111}
h1{font-size:1.25rem}
.empty{color:#666}
table{border-collapse:collapse;width:100%}
th,td{text-align:left;border-bottom:1px solid #ddd;padding:.4rem .3rem;vertical-align:top}
th{color:#555;font-weight:600}
code{font-size:.85rem}
</style>
<h1>Входящая почта</h1>
{{if not .Messages}}
<p class="empty">Писем нет. CloudMailin ещё не стучал.</p>
{{else}}
<p>Писем: {{len .Messages}}</p>
<table>
<tr><th>Когда</th><th>От</th><th>Тема</th><th>Файлы</th></tr>
{{range $m := .Messages}}
<tr>
<td><code>{{or $m.Received $m.ID}}</code></td>
<td>{{or $m.From "—"}}</td>
<td>{{or $m.Subject "без темы"}}</td>
<td>
{{range $m.Files}}<div><a href="/m/{{$m.ID}}/{{.}}{{$.Query}}">{{.}}</a></div>{{else}}—{{end}}
</td>
</tr>
{{end}}
</table>
{{end}}
`))
