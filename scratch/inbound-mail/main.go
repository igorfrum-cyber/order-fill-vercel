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
	receive := gate(token, func(w http.ResponseWriter, r *http.Request) {
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
	})
	mux.HandleFunc("POST /inbound", receive)
	mux.HandleFunc("POST /{$}", receive)
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
	mux.HandleFunc("GET /m/{id}", gate(token, func(w http.ResponseWriter, r *http.Request) {
		id := safeName(r.PathValue("id"))
		if id == "unnamed" {
			http.NotFound(w, r)
			return
		}
		msg, err := loadMessage(dir, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := messagePage{Token: r.URL.Query().Get("token"), Message: msg}
		if err := messageTmpl.Execute(w, page); err != nil {
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
	if user, pass, ok := r.BasicAuth(); ok {
		if pass != "" {
			return pass
		}
		return user
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
	Plain    string   `json:"-"`
	HTML     string   `json:"-"`
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

func formFirst(values map[string][]string, keys ...string) string {
	for _, key := range keys {
		if xs := values[key]; len(xs) > 0 && strings.TrimSpace(xs[0]) != "" {
			return xs[0]
		}
	}
	return ""
}

func summarizeForm(values map[string][]string) mailSummary {
	return mailSummary{
		From:    formFirst(values, "envelope[from]", "headers[from]"),
		To:      formFirst(values, "envelope[to]", "headers[to]"),
		Subject: formFirst(values, "headers[subject]"),
	}
}

func writeMeta(box *os.Root, sum mailSummary) error {
	raw, err := json.MarshalIndent(sum, "", "  ")
	if err != nil {
		return err
	}
	return writeExclusive(box, "meta.json", raw)
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
		files, err := dumpMultipart(box, r)
		if err != nil {
			return dir, err
		}
		sum := summarizeForm(r.MultipartForm.Value)
		sum.ID = stamp
		sum.Received = time.Now().UTC().Format(time.RFC3339)
		if len(files) > 0 {
			sum.Files = files
		}
		return dir, writeMeta(box, sum)
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
	return dir, writeMeta(box, sum)
}

func dumpMultipart(box *os.Root, r *http.Request) ([]string, error) {
	form := r.MultipartForm
	if form == nil {
		return nil, nil
	}
	values, err := json.MarshalIndent(form.Value, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := writeExclusive(box, "form.json", values); err != nil {
		return nil, err
	}
	if plain := formFirst(form.Value, "plain"); plain != "" {
		if err := writeExclusive(box, "plain.txt", []byte(plain)); err != nil {
			return nil, err
		}
	}
	if htmlBody := formFirst(form.Value, "html"); htmlBody != "" {
		if err := writeExclusive(box, "html.txt", []byte(htmlBody)); err != nil {
			return nil, err
		}
	}
	n := 0
	var saved []string
	for _, fhs := range form.File {
		for _, fh := range fhs {
			src, err := fh.Open()
			if err != nil {
				return saved, err
			}
			data, err := io.ReadAll(io.LimitReader(src, maxBody+1))
			_ = src.Close()
			if err != nil {
				return saved, err
			}
			if len(data) > maxBody {
				return saved, errors.New("attachment too large")
			}
			n++
			name := fmt.Sprintf("%02d-%s", n, safeName(fh.Filename))
			if err := writeExclusive(box, name, data); err != nil {
				return saved, err
			}
			saved = append(saved, name)
		}
	}
	return saved, nil
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
		out = append(out, loadSummary(filepath.Join(root, ent.Name())))
	}
	slices.SortFunc(out, func(a, b mailSummary) int {
		return strings.Compare(b.ID, a.ID)
	})
	return out, nil
}

var sidecarFiles = map[string]bool{
	"meta.json": true, "headers.txt": true, "body": true, "form.json": true, "plain.txt": true, "html.txt": true,
}

func loadSummary(folder string) mailSummary {
	id := filepath.Base(folder)
	sum := mailSummary{ID: id}
	if raw, err := os.ReadFile(filepath.Join(folder, "meta.json")); err == nil {
		_ = json.Unmarshal(raw, &sum)
		sum.ID = id
	}
	if sum.From == "" || sum.Subject == "" {
		if raw, err := os.ReadFile(filepath.Join(folder, "form.json")); err == nil {
			var form map[string][]string
			if json.Unmarshal(raw, &form) == nil {
				got := summarizeForm(form)
				if sum.From == "" {
					sum.From = got.From
				}
				if sum.To == "" {
					sum.To = got.To
				}
				if sum.Subject == "" {
					sum.Subject = got.Subject
				}
			}
		}
	}
	if len(sum.Files) == 0 {
		sum.Files = listedAttachments(folder)
	}
	return sum
}

func listedAttachments(folder string) []string {
	ents, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}
	var out []string
	for _, ent := range ents {
		if ent.IsDir() || sidecarFiles[ent.Name()] {
			continue
		}
		out = append(out, ent.Name())
	}
	return out
}

func loadMessage(root, id string) (mailSummary, error) {
	folder := filepath.Join(root, id)
	st, err := os.Stat(folder)
	if err != nil || !st.IsDir() {
		return mailSummary{}, os.ErrNotExist
	}
	sum := loadSummary(folder)
	if raw, err := os.ReadFile(filepath.Join(folder, "plain.txt")); err == nil {
		sum.Plain = string(raw)
	} else if raw, err := os.ReadFile(filepath.Join(folder, "form.json")); err == nil {
		var form map[string][]string
		if json.Unmarshal(raw, &form) == nil {
			sum.Plain = formFirst(form, "plain")
			sum.HTML = formFirst(form, "html")
		}
	}
	if sum.HTML == "" {
		if raw, err := os.ReadFile(filepath.Join(folder, "html.txt")); err == nil {
			sum.HTML = string(raw)
		}
	}
	return sum, nil
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

type messagePage struct {
	Token   string
	Message mailSummary
}

func (p inboxPage) Query() string {
	if p.Token == "" {
		return ""
	}
	return "?token=" + p.Token
}

func (p messagePage) Query() string {
	return inboxPage{Token: p.Token}.Query()
}

var pageCSS = `
body{font-family:sans-serif;max-width:56rem;margin:2rem auto;padding:0 1rem;color:#111}
h1{font-size:1.25rem}
.empty{color:#666}
table{border-collapse:collapse;width:100%}
th,td{text-align:left;border-bottom:1px solid #ddd;padding:.4rem .3rem;vertical-align:top}
th{color:#555;font-weight:600}
code,pre{font-size:.85rem}
pre{white-space:pre-wrap;background:#f6f6f6;padding:.75rem;overflow:auto}
a{color:#06c}
.meta{color:#555;margin:.3rem 0}
`

var inboxTmpl = template.Must(template.New("inbox").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Входящая почта</title>
<style>` + pageCSS + `</style>
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
<td><a href="/m/{{$m.ID}}{{$.Query}}">{{or $m.Subject "без темы"}}</a></td>
<td>
{{range $m.Files}}<div><a href="/m/{{$m.ID}}/{{.}}{{$.Query}}">{{.}}</a></div>{{else}}—{{end}}
</td>
</tr>
{{end}}
</table>
{{end}}
`))

var messageTmpl = template.Must(template.New("message").Parse(`<!doctype html>
<meta charset="utf-8">
<title>{{or .Message.Subject "Письмо"}}</title>
<style>` + pageCSS + `</style>
<p><a href="/{{.Query}}">← все письма</a></p>
<h1>{{or .Message.Subject "без темы"}}</h1>
<p class="meta">От: {{or .Message.From "—"}}<br>Кому: {{or .Message.To "—"}}<br>Когда: <code>{{or .Message.Received .Message.ID}}</code></p>
{{if .Message.Files}}
<p>Файлы:{{range .Message.Files}} <a href="/m/{{$.Message.ID}}/{{.}}{{$.Query}}">{{.}}</a>{{end}}</p>
{{end}}
{{if .Message.Plain}}
<h2>Текст</h2>
<pre>{{.Message.Plain}}</pre>
{{end}}
{{if .Message.HTML}}
<h2>HTML</h2>
<pre>{{.Message.HTML}}</pre>
{{end}}
{{if and (not .Message.Plain) (not .Message.HTML)}}
<p class="empty">Тела письма нет — только заголовки и вложения.</p>
{{end}}
`))
