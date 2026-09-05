package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

const (
	sessionCookieName = "order_fill_session"
	authJSONLimit     = 8 << 10
	jobJSONLimit      = 1 << 20
)

type userContextKey struct{}

type User struct {
	ID          string
	Login       string
	Role        string
	CompanyID   string
	CompanyName string
	LoginSlug   string
	HasLogo     bool
	TwoFactor   bool
	HasPasskey  bool
}

func userFromProto(u *identityv1.User) User {
	if u == nil {
		return User{}
	}
	return User{
		ID: u.GetId(), Login: u.GetLogin(), Role: u.GetRole(), CompanyID: u.GetCompanyId(),
		CompanyName: u.GetCompanyName(), LoginSlug: u.GetLoginSlug(), HasLogo: u.GetHasLogo(),
		TwoFactor: u.GetTwoFactorEnabled(), HasPasskey: u.GetHasPasskey(),
	}
}

func withUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func userFrom(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(userContextKey{}).(User)
	return user, ok
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": publicErrorMessage(code, message)})
}

func writeGRPCError(w http.ResponseWriter, fallback string, err error) {
	st := status.Convert(err)
	switch st.Code() {
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
	case codes.NotFound:
		writeError(w, http.StatusNotFound, "not_found", st.Message())
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, "conflict", st.Message())
	case codes.InvalidArgument, codes.FailedPrecondition:
		writeError(w, http.StatusBadRequest, "bad_request", st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusNotFound, "not_found", "not found")
	default:
		writeError(w, http.StatusInternalServerError, fallback, st.Message())
	}
}

func writeSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool, domain string) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 8 * 3600
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", Domain: domain,
		MaxAge: maxAge, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", Domain: domain,
		MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure,
	})
}

func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	defer func() { _ = r.Body.Close() }()
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return false
	}
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func contentDisposition(name string) string {
	ascii := strings.Map(func(char rune) rune {
		if char < 32 || char > 126 || char == '"' || char == '\\' {
			return '_'
		}
		return char
	}, name)
	return `attachment; filename="` + ascii + `"; filename*=UTF-8''` + url.PathEscape(name)
}

func csrfAllowed(r *http.Request, allowedOrigins []string) bool {
	if r.Header.Get("X-Requested-With") != "fetch" {
		return false
	}
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin == "" {
		return true
	}
	return originAllowed(origin, allowedOrigins)
}

func originAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if originsMatch(strings.TrimRight(allowed, "/"), origin) {
			return true
		}
	}
	return false
}

func originsMatch(allowed string, origin string) bool {
	if strings.EqualFold(allowed, origin) {
		return true
	}
	allowedURL, err := url.Parse(allowed)
	if err != nil || allowedURL.Scheme == "" || allowedURL.Host == "" {
		return false
	}
	originURL, err := url.Parse(origin)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" {
		return false
	}
	if !strings.EqualFold(allowedURL.Scheme, originURL.Scheme) {
		return false
	}
	if originPort(allowedURL) != originPort(originURL) {
		return false
	}
	if isLoopbackHost(allowedURL.Hostname()) && isLoopbackHost(originURL.Hostname()) {
		return true
	}
	return hostIsCompanySubdomain(originURL.Hostname(), allowedURL.Hostname())
}

func originPort(parsed *url.URL) string {
	if port := parsed.Port(); port != "" {
		return port
	}
	if parsed.Scheme == "https" {
		return "443"
	}
	return "80"
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	if strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

var companySlugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

func hostIsCompanySubdomain(originHost string, allowedHost string) bool {
	originHost = strings.ToLower(originHost)
	allowedHost = strings.ToLower(allowedHost)
	suffix := "." + allowedHost
	if !strings.HasSuffix(originHost, suffix) {
		return false
	}
	slug := strings.TrimSuffix(originHost, suffix)
	return slug != "" && !strings.Contains(slug, ".") && companySlugPattern.MatchString(slug)
}

func ParseAllowedOrigins(value string) []string {
	origins := make([]string, 0)
	for _, entry := range strings.Split(value, ",") {
		trimmed := strings.TrimRight(strings.TrimSpace(entry), "/")
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			return []string{"*"}
		}
		origins = append(origins, trimmed)
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, allowed := range allowedOrigins {
				if allowed == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
					break
				}
				if originAllowed(origin, []string{allowed}) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					break
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func publicCompanyLoginPath(path string) bool {
	const prefix = "/api/v1/public/companies/"
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return false
	}
	slug, suffix, found := strings.Cut(rest, "/")
	return found && slug != "" && (suffix == "login" || suffix == "logo")
}

func setSecurityHeaders(w http.ResponseWriter) {
	header := w.Header()
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Referrer-Policy", "no-referrer")
}

func publicErrorMessage(code, message string) string {
	switch code {
	case "unauthorized":
		return "Нужно войти заново."
	case "not_found":
		return "Не нашли то, что искали."
	case "conflict":
		return "Такая запись уже есть."
	}
	if message != "" {
		return message
	}
	return "Что-то пошло не так. Попробуйте ещё раз."
}
