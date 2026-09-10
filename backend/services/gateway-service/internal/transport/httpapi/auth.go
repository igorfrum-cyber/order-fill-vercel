package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	passkeyv1 "order-fill/backend/proto/gen/go/orderfill/passkey/v1"
	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
)

type authJSON struct {
	Login           string          `json:"login"`
	Password        string          `json:"password"`
	CurrentPassword string          `json:"current_password"`
	Token           string          `json:"token"`
	ChallengeID     string          `json:"challenge_id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Credential      json.RawMessage `json:"credential"`
}

func presentUser(user User) map[string]any {
	out := map[string]any{
		"id": user.ID, "login": user.Login, "role": user.Role,
		"company_id": user.CompanyID, "company_name": user.CompanyName, "login_slug": user.LoginSlug,
		"has_logo": user.HasLogo, "two_factor_enabled": user.TwoFactor, "has_passkey": user.HasPasskey,
	}
	return out
}

func (a *API) presentUser(ctx context.Context, user User) map[string]any {
	if user.CompanyID != "" {
		user.HasLogo = a.companyHasLogo(ctx, user.CompanyID)
	}
	return presentUser(user)
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var payload authJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.Identity.Login(r.Context(), &identityv1.LoginRequest{Login: payload.Login, Password: payload.Password})
	if err != nil {
		writeGRPCError(w, "login_failed", err)
		return
	}
	if resp.GetTwoFactorRequired() {
		writeJSON(w, http.StatusOK, map[string]any{"two_factor_required": true, "challenge_id": resp.GetChallengeId()})
		return
	}
	a.writeLogin(w, r, resp.GetSession(), resp.GetUser())
}

func (a *API) login2FA(w http.ResponseWriter, r *http.Request) {
	var payload authJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.Identity.CompleteTwoFactorLogin(r.Context(), &identityv1.CompleteTwoFactorLoginRequest{
		ChallengeId: payload.ChallengeID, Code: payload.Code,
	})
	if err != nil {
		writeGRPCError(w, "login_failed", err)
		return
	}
	a.writeLogin(w, r, resp.GetSession(), resp.GetUser())
}

func (a *API) invite(w http.ResponseWriter, r *http.Request) {
	var payload authJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.Identity.AcceptInvite(r.Context(), &identityv1.AcceptInviteRequest{Token: payload.Token, Password: payload.Password})
	if err != nil {
		writeGRPCError(w, "invite_failed", err)
		return
	}
	a.writeLogin(w, r, resp.GetSession(), resp.GetUser())
}

func (a *API) writeLogin(w http.ResponseWriter, r *http.Request, session *identityv1.Session, user *identityv1.User) {
	writeSessionCookie(w, session.GetToken(), time.Now().Add(8*time.Hour), a.CookieSecure, a.CookieDomain)
	writeJSON(w, http.StatusOK, a.presentUser(r.Context(), userFromProto(user)))
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if token := sessionToken(r); token != "" {
		_, _ = a.Clients.Identity.Logout(r.Context(), &identityv1.LogoutRequest{SessionToken: token})
	}
	clearSessionCookie(w, a.CookieSecure, a.CookieDomain)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) logoutEverywhere(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	_, err := a.Clients.Identity.LogoutEverywhere(r.Context(), &identityv1.LogoutEverywhereRequest{
		ActorUserId: user.ID, SessionToken: sessionToken(r),
	})
	if err != nil {
		writeGRPCError(w, "logout_everywhere_failed", err)
		return
	}
	clearSessionCookie(w, a.CookieSecure, a.CookieDomain)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listSessions(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	resp, err := a.Clients.Identity.ListSessions(r.Context(), &identityv1.ListSessionsRequest{
		ActorUserId: user.ID, SessionToken: sessionToken(r),
	})
	if err != nil {
		writeGRPCError(w, "session_list_failed", err)
		return
	}
	items := make([]map[string]any, 0, len(resp.GetSessions()))
	for _, item := range resp.GetSessions() {
		items = append(items, map[string]any{
			"id": item.GetId(), "device": item.GetDevice(), "current": item.GetCurrent(),
			"created_at": item.GetCreatedAt(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": items})
}

func (a *API) revokeSession(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	_, err := a.Clients.Identity.RevokeSession(r.Context(), &identityv1.RevokeSessionRequest{
		ActorUserId: user.ID, SessionToken: sessionToken(r), SessionId: r.PathValue("id"),
	})
	if err != nil {
		writeGRPCError(w, "session_revoke_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) changePassword(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload authJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	_, err := a.Clients.Identity.ChangePassword(r.Context(), &identityv1.ChangePasswordRequest{
		ActorUserId: user.ID, CurrentPassword: payload.CurrentPassword, NewPassword: payload.Password,
	})
	if err != nil {
		writeGRPCError(w, "change_password_failed", err)
		return
	}
	a.recordAudit(r.Context(), user, "password_changed", user.CompanyID, user.CompanyName)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	writeJSON(w, http.StatusOK, a.presentUser(r.Context(), user))
}

func (a *API) totpSetup(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	resp, err := a.Clients.TwoFA.Setup(r.Context(), &twofav1.SetupRequest{ActorUserId: user.ID, AccountName: user.Login})
	if err != nil {
		writeGRPCError(w, "totp_setup_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"secret": resp.GetSecret(), "otpauth_url": resp.GetOtpauthUrl(),
		"qr_png_base64": base64.StdEncoding.EncodeToString(resp.GetQrPng()),
	})
}

func (a *API) totpEnable(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload authJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.TwoFA.Enable(r.Context(), &twofav1.EnableRequest{ActorUserId: user.ID, Code: payload.Code})
	if err != nil {
		writeGRPCError(w, "totp_enable_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": resp.GetRecoveryCodes()})
}

func (a *API) totpDisable(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload authJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.Identity.Login(r.Context(), &identityv1.LoginRequest{Login: user.Login, Password: payload.Password})
	if err != nil {
		writeGRPCError(w, "totp_disable_failed", err)
		return
	}
	if session := resp.GetSession(); session.GetToken() != "" {
		_, _ = a.Clients.Identity.Logout(r.Context(), &identityv1.LogoutRequest{SessionToken: session.GetToken()})
	}
	_, err = a.Clients.TwoFA.Disable(r.Context(), &twofav1.DisableRequest{ActorUserId: user.ID})
	if err != nil {
		writeGRPCError(w, "totp_disable_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) passkeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	resp, err := a.Clients.Passkey.BeginRegistration(r.Context(), &passkeyv1.BeginRegistrationRequest{ActorUserId: user.ID, Origin: origin})
	if err != nil {
		writeGRPCError(w, "passkey_begin_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"challenge_id": resp.GetChallengeId(), "options": json.RawMessage(resp.GetOptionsJson())})
}

func (a *API) passkeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload authJSON
	if !decodeJSON(w, r, &payload, jobJSONLimit) {
		return
	}
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	_, err := a.Clients.Passkey.FinishRegistration(r.Context(), &passkeyv1.FinishRegistrationRequest{
		ActorUserId: user.ID, ChallengeId: payload.ChallengeID, CredentialJson: payload.Credential, Origin: origin,
	})
	if err != nil {
		writeGRPCError(w, "passkey_finish_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) passkeyList(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	resp, err := a.Clients.Passkey.ListCredentials(r.Context(), &passkeyv1.ListCredentialsRequest{ActorUserId: user.ID})
	if err != nil {
		writeGRPCError(w, "passkey_list_failed", err)
		return
	}
	items := make([]map[string]any, 0, len(resp.GetCredentials()))
	for _, item := range resp.GetCredentials() {
		items = append(items, map[string]any{"id": item.GetId(), "name": item.GetName(), "created_at": item.GetCreatedAt()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"passkeys": items})
}

func (a *API) passkeyDelete(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	_, err := a.Clients.Passkey.DeleteCredential(r.Context(), &passkeyv1.DeleteCredentialRequest{
		ActorUserId: user.ID, CredentialId: r.PathValue("id"),
	})
	if err != nil {
		writeGRPCError(w, "passkey_delete_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) passkeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	var payload authJSON
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	resp, err := a.Clients.Passkey.BeginLogin(r.Context(), &passkeyv1.BeginLoginRequest{Login: payload.Login, Origin: origin})
	if err != nil {
		writeGRPCError(w, "passkey_begin_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"challenge_id": resp.GetChallengeId(), "options": json.RawMessage(resp.GetOptionsJson())})
}

func (a *API) passkeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	var payload authJSON
	if !decodeJSON(w, r, &payload, jobJSONLimit) {
		return
	}
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	resp, err := a.Clients.Identity.FinishPasskeyLogin(r.Context(), &identityv1.FinishPasskeyLoginRequest{
		ChallengeId: payload.ChallengeID, CredentialJson: payload.Credential, Origin: origin,
	})
	if err != nil {
		writeGRPCError(w, "passkey_finish_failed", err)
		return
	}
	a.writeLogin(w, r, resp.GetSession(), resp.GetUser())
}
