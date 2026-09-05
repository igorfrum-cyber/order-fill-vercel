package httpapi

import (
	"context"
	"net/http"
	"time"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/gateway-service/internal/clients"
	"order-fill/backend/services/gateway-service/internal/config"
)

type API struct {
	Clients        clients.Clients
	HTTP           *http.Client
	IdentityHTTP   string
	WorkerHealth   string
	FileHealth     string
	PostgresAddr   string
	RedisAddr      string
	AllowedOrigins []string
	CookieSecure   bool
	CookieDomain   string
}

func New(cfg config.Config, c clients.Clients) http.Handler {
	api := &API{
		Clients:        c,
		HTTP:           &http.Client{Timeout: 2 * time.Second},
		IdentityHTTP:   cfg.IdentityHTTP,
		WorkerHealth:   cfg.WorkerHealth,
		FileHealth:     cfg.FileHealth,
		PostgresAddr:   cfg.PostgresAddr,
		RedisAddr:      cfg.RedisAddr,
		AllowedOrigins: ParseAllowedOrigins(cfg.AllowedOrigins),
		CookieSecure:   cfg.CookieSecure,
		CookieDomain:   cfg.CookieDomain,
	}
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	mux.HandleFunc("POST /api/v1/auth/login", api.login)
	mux.HandleFunc("POST /api/v1/auth/login/2fa", api.login2FA)
	mux.HandleFunc("POST /api/v1/auth/invite", api.invite)
	mux.HandleFunc("POST /api/v1/auth/logout", api.logout)
	mux.HandleFunc("POST /api/v1/auth/logout-everywhere", api.logoutEverywhere)
	mux.HandleFunc("GET /api/v1/auth/sessions", api.listSessions)
	mux.HandleFunc("POST /api/v1/auth/sessions/{id}/delete", api.revokeSession)
	mux.HandleFunc("POST /api/v1/auth/password", api.changePassword)
	mux.HandleFunc("GET /api/v1/auth/me", api.me)
	mux.HandleFunc("POST /api/v1/auth/2fa/setup", api.totpSetup)
	mux.HandleFunc("POST /api/v1/auth/2fa/enable", api.totpEnable)
	mux.HandleFunc("POST /api/v1/auth/2fa/disable", api.totpDisable)
	mux.HandleFunc("POST /api/v1/auth/passkeys/register/begin", api.passkeyRegisterBegin)
	mux.HandleFunc("POST /api/v1/auth/passkeys/register/finish", api.passkeyRegisterFinish)
	mux.HandleFunc("GET /api/v1/auth/passkeys", api.passkeyList)
	mux.HandleFunc("POST /api/v1/auth/passkeys/{id}/delete", api.passkeyDelete)
	mux.HandleFunc("POST /api/v1/auth/passkeys/login/begin", api.passkeyLoginBegin)
	mux.HandleFunc("POST /api/v1/auth/passkeys/login/finish", api.passkeyLoginFinish)
	mux.HandleFunc("POST /api/v1/jobs/order-fill", api.createOrderFill)
	mux.HandleFunc("POST /api/v1/jobs/north-merge", api.createNorthMerge)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", api.getJob)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/report", api.getReport)
	mux.HandleFunc("POST /api/v1/jobs/{job_id}/edits", api.submitEdits)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files", api.listFiles)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/archive", api.downloadArchive)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}", api.downloadFile)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}/preview", api.previewMeta)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}/preview/window", api.previewWindow)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}/preview/find", api.previewFind)
	mux.HandleFunc("GET /api/v1/jobs", api.listJobs)
	mux.HandleFunc("GET /api/v1/companies", api.listCompanies)
	mux.HandleFunc("POST /api/v1/companies", api.createCompany)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/disable", api.disableCompany)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/login-slug", api.updateCompany)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/profile", api.updateCompany)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/logo", api.setCompanyLogo)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/logo/clear", api.clearCompanyLogo)
	mux.HandleFunc("GET /api/v1/companies/{company_id}/users", api.listUsers)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/users", api.createUser)
	mux.HandleFunc("POST /api/v1/users/{user_id}/disable", api.disableUser)
	mux.HandleFunc("POST /api/v1/users/{user_id}/reset", api.resetUser)
	mux.HandleFunc("GET /api/v1/audit", api.listAudit)
	mux.HandleFunc("GET /api/v1/status", api.listStatus)
	mux.HandleFunc("GET /api/v1/public/companies/{slug}/login", api.publicCompanyLogin)
	mux.HandleFunc("GET /api/v1/public/companies/{slug}/logo", api.publicCompanyLogo)
	return withCORS(api.gate(mux), api.AllowedOrigins)
}

func (a *API) gate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/login/2fa" || r.URL.Path == "/api/v1/auth/invite" ||
			r.URL.Path == "/api/v1/auth/passkeys/login/begin" || r.URL.Path == "/api/v1/auth/passkeys/login/finish" ||
			publicCompanyLoginPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost && !csrfAllowed(r, a.AllowedOrigins) {
			writeError(w, http.StatusForbidden, "forbidden", "request was rejected")
			return
		}
		token := sessionToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		resp, err := a.Clients.Identity.ValidateSession(r.Context(), &identityv1.ValidateSessionRequest{SessionToken: token})
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), userFromProto(resp.GetUser()))))
	})
}

func (a *API) meta(user User) *commonv1.RequestMeta {
	return &commonv1.RequestMeta{ActorUserId: user.ID, CompanyId: user.CompanyID, RequestId: grpcutil.NewID()}
}

func (a *API) jobCtx(r *http.Request, user User) context.Context {
	return grpcutil.WithActorRole(r.Context(), user.Role)
}
