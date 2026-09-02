package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"order-fill/services/api-service/internal/app/usecase"
	"order-fill/services/api-service/internal/domain/identity"
)

const passkeyJSONLimit = 64 << 10

type passkeyAPI interface {
	BeginPasskeyRegistration(ctx context.Context, actor identity.User, origin string, name string) (usecase.PasskeyBegin, error)
	FinishPasskeyRegistration(ctx context.Context, actor identity.User, origin string, challengeID string, response json.RawMessage, name string) (identity.PasskeyPublicView, error)
	ListPasskeys(ctx context.Context, actor identity.User) ([]identity.PasskeyPublicView, error)
	DeletePasskey(ctx context.Context, actor identity.User, id string) error
	BeginPasskeyLogin(ctx context.Context, origin string, login string) (usecase.PasskeyBegin, error)
	FinishPasskeyLogin(ctx context.Context, origin string, challengeID string, response json.RawMessage) (usecase.Session, error)
}

type passkeyHandler struct {
	auth           passkeyAPI
	cookieSecure   bool
	cookieDomain   string
	loginLimiter   *Limiter
	allowedOrigins []string
}

func (h passkeyHandler) registerBegin(w http.ResponseWriter, r *http.Request) {
	user, origin, ok := h.ready(w, r, true)
	if !ok {
		return
	}
	payload, decoded := decodePasskeyJSON(w, r)
	if !decoded {
		return
	}
	begin, err := h.auth.BeginPasskeyRegistration(r.Context(), user, origin, payload.Name)
	if err != nil {
		writeDomainError(w, "passkey_register_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, passkeyBeginResponse{ChallengeID: begin.ChallengeID, Options: begin.Options})
}

func (h passkeyHandler) registerFinish(w http.ResponseWriter, r *http.Request) {
	user, origin, ok := h.ready(w, r, true)
	if !ok {
		return
	}
	payload, decoded := decodePasskeyJSON(w, r)
	if !decoded {
		return
	}
	view, err := h.auth.FinishPasskeyRegistration(r.Context(), user, origin, payload.ChallengeID, payload.Credential, payload.Name)
	if err != nil {
		writeDomainError(w, "passkey_register_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h passkeyHandler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok || h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	items, err := h.auth.ListPasskeys(r.Context(), user)
	if err != nil {
		writeDomainError(w, "passkey_list_failed", err)
		return
	}
	if items == nil {
		items = []identity.PasskeyPublicView{}
	}
	writeJSON(w, http.StatusOK, passkeyListResponse{Passkeys: items})
}

func (h passkeyHandler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok || h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if err := h.auth.DeletePasskey(r.Context(), user, r.PathValue("id")); err != nil {
		writeDomainError(w, "passkey_delete_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h passkeyHandler) loginBegin(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	origin, ok := h.origin(w, r)
	if !ok {
		return
	}
	payload, decoded := decodePasskeyJSON(w, r)
	if !decoded {
		return
	}
	key := clientIP(r) + "\x00passkey-begin\x00" + payload.Login
	if h.loginLimiter != nil && !h.loginLimiter.Allow(key) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
		return
	}
	begin, err := h.auth.BeginPasskeyLogin(r.Context(), origin, payload.Login)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	writeJSON(w, http.StatusOK, passkeyBeginResponse{ChallengeID: begin.ChallengeID, Options: begin.Options})
}

func (h passkeyHandler) loginFinish(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	origin, ok := h.origin(w, r)
	if !ok {
		return
	}
	payload, decoded := decodePasskeyJSON(w, r)
	if !decoded {
		return
	}
	key := clientIP(r) + "\x00passkey-finish\x00" + payload.ChallengeID
	if h.loginLimiter != nil && !h.loginLimiter.Allow(key) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
		return
	}
	session, err := h.auth.FinishPasskeyLogin(requestClient(r), origin, payload.ChallengeID, payload.Credential)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	writeSessionCookie(w, session, h.cookieSecure, h.cookieDomain)
	writeJSON(w, http.StatusOK, presentUser(session.User))
}

func (h passkeyHandler) ready(w http.ResponseWriter, r *http.Request, requireUser bool) (identity.User, string, bool) {
	var user identity.User
	if requireUser {
		got, ok := userFrom(r)
		if !ok || h.auth == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return identity.User{}, "", false
		}
		user = got
	}
	origin, ok := h.origin(w, r)
	if !ok {
		return identity.User{}, "", false
	}
	return user, origin, true
}

func (h passkeyHandler) origin(w http.ResponseWriter, r *http.Request) (string, bool) {
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin == "" || !originAllowed(origin, h.allowedOrigins) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid origin")
		return "", false
	}
	return origin, true
}

type passkeyJSON struct {
	Login       string          `json:"login"`
	Name        string          `json:"name"`
	ChallengeID string          `json:"challenge_id"`
	Credential  json.RawMessage `json:"credential"`
}

func decodePasskeyJSON(w http.ResponseWriter, r *http.Request) (passkeyJSON, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, passkeyJSONLimit)
	defer func() { _ = r.Body.Close() }()
	var payload passkeyJSON
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return passkeyJSON{}, false
	}
	return payload, true
}

type passkeyBeginResponse struct {
	ChallengeID string          `json:"challenge_id"`
	Options     json.RawMessage `json:"options"`
}

type passkeyListResponse struct {
	Passkeys []identity.PasskeyPublicView `json:"passkeys"`
}
