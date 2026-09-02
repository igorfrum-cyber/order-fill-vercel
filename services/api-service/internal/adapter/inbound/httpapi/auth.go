package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"order-fill/services/api-service/internal/app/usecase"
	"order-fill/services/api-service/internal/domain/identity"
)

const (
	sessionCookieName = "order_fill_session"
	authJSONLimit     = 8 << 10
)

type userContextKey struct{}

type sessionAuthenticator interface {
	Login(ctx context.Context, login string, password string) (usecase.Session, error)
	Logout(ctx context.Context, tokenHash string) error
	LogoutEverywhere(ctx context.Context, actor identity.User) error
	AcceptInvite(ctx context.Context, rawToken string, password string) (usecase.Session, error)
	ChangePassword(ctx context.Context, actor identity.User, current string, next string) error
	SessionUser(ctx context.Context, tokenHash string) (identity.User, error)
}

func withUser(ctx context.Context, user identity.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func userFrom(r *http.Request) (identity.User, bool) {
	user, ok := r.Context().Value(userContextKey{}).(identity.User)
	return user, ok
}

func writeSessionCookie(w http.ResponseWriter, session usecase.Session, secure bool) {
	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.RawToken,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func sessionTokenHash(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}
	return identity.HashSecret(cookie.Value)
}

type gate struct {
	next           http.Handler
	auth           sessionAuthenticator
	allowedOrigins []string
	loginLimiter   *Limiter
	createLimiter  *Limiter
}

func (g gate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	if r.URL.Path == "/healthz" {
		g.next.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/invite" {
		g.next.ServeHTTP(w, r)
		return
	}
	if r.Method == http.MethodPost && !csrfAllowed(r, g.allowedOrigins) {
		writeError(w, http.StatusForbidden, "forbidden", "request was rejected")
		return
	}
	if g.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	hash := sessionTokenHash(r)
	if hash == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	user, err := g.auth.SessionUser(r.Context(), hash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if r.URL.Path == "/metrics" && user.Role != identity.RolePlatformAdmin {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/jobs/") &&
		(strings.HasSuffix(r.URL.Path, "/order-fill") || strings.HasSuffix(r.URL.Path, "/north-merge")) {
		if g.createLimiter != nil && !g.createLimiter.Allow(user.ID) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
	}
	g.next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
}

func setSecurityHeaders(w http.ResponseWriter) {
	header := w.Header()
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
}

type authHandler struct {
	auth         sessionAuthenticator
	cookieSecure bool
	loginLimiter *Limiter
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	payload, ok := decodeAuthJSON(w, r)
	if !ok {
		return
	}
	key := clientIP(r) + "\x00" + payload.Login
	if h.loginLimiter != nil && !h.loginLimiter.Allow(key) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
		return
	}
	session, err := h.auth.Login(r.Context(), payload.Login, payload.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	writeSessionCookie(w, session, h.cookieSecure)
	writeJSON(w, http.StatusOK, presentUser(session.User))
}

func (h authHandler) invite(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	payload, ok := decodeAuthJSON(w, r)
	if !ok {
		return
	}
	session, err := h.auth.AcceptInvite(r.Context(), payload.Token, payload.Password)
	if err != nil {
		writeDomainError(w, "invite_failed", err)
		return
	}
	writeSessionCookie(w, session, h.cookieSecure)
	writeJSON(w, http.StatusOK, presentUser(session.User))
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if hash := sessionTokenHash(r); hash != "" && h.auth != nil {
		_ = h.auth.Logout(r.Context(), hash)
	}
	clearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) logoutEverywhere(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok || h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if err := h.auth.LogoutEverywhere(r.Context(), user); err != nil {
		writeDomainError(w, "logout_everywhere_failed", err)
		return
	}
	clearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok || h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	payload, decoded := decodeAuthJSON(w, r)
	if !decoded {
		return
	}
	if err := h.auth.ChangePassword(r.Context(), user, payload.CurrentPassword, payload.Password); err != nil {
		writeDomainError(w, "change_password_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	writeJSON(w, http.StatusOK, presentUser(user))
}

type authJSON struct {
	Login           string `json:"login"`
	Password        string `json:"password"`
	CurrentPassword string `json:"current_password"`
	Token           string `json:"token"`
}

func decodeAuthJSON(w http.ResponseWriter, r *http.Request) (authJSON, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, authJSONLimit)
	defer func() { _ = r.Body.Close() }()
	var payload authJSON
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if err != io.EOF {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
			return authJSON{}, false
		}
	}
	return payload, true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type userResponse struct {
	ID          string  `json:"id"`
	Login       string  `json:"login"`
	Role        string  `json:"role"`
	CompanyID   string  `json:"company_id,omitempty"`
	CompanyName string  `json:"company_name,omitempty"`
	DisabledAt  *string `json:"disabled_at,omitempty"`
}

func presentUser(user identity.User) userResponse {
	response := userResponse{
		ID:          user.ID,
		Login:       user.Login,
		Role:        string(user.Role),
		CompanyID:   user.CompanyID,
		CompanyName: user.CompanyName,
	}
	if user.DisabledAt != nil {
		value := user.DisabledAt.UTC().Format("2006-01-02T15:04:05Z")
		response.DisabledAt = &value
	}
	return response
}
