package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"order-fill/services/api-service/internal/domain/identity"
)

// MetricsReader exposes a counters snapshot for the /metrics endpoint.
type MetricsReader interface {
	Snapshot() map[string]int64
}

type accessResetter interface {
	ResetAccess(ctx context.Context, actor identity.User, userID string) (string, error)
}

// Config wires the router. Every dependency is optional so the router can be
// built in tests with only the handlers under test.
type Config struct {
	CreateJob       jobCreator
	GetJob          jobFinder
	GetReport       reportFinder
	ListFiles       fileLister
	DownloadFile    fileDownloader
	DownloadArchive archiveDownloader
	SubmitEdits     editSubmitter
	Preview         previewReader
	ListJobs        lister
	Auth            sessionAuthenticator
	Admin           adminAPI
	Reset           accessResetter
	Metrics         MetricsReader
	AllowedOrigins  []string
	MaxUploadBytes  int64
	CookieSecure    bool
	LoginLimiter    *Limiter
	CreateLimiter   *Limiter
}

func NewRouter(config Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler{})
	if config.Metrics != nil {
		mux.Handle("GET /metrics", metricsHandler{metrics: config.Metrics})
	}

	auth := authHandler{auth: config.Auth, cookieSecure: config.CookieSecure, loginLimiter: config.LoginLimiter}
	mux.HandleFunc("POST /api/v1/auth/login", auth.login)
	mux.HandleFunc("POST /api/v1/auth/invite", auth.invite)
	mux.HandleFunc("POST /api/v1/auth/logout", auth.logout)
	mux.HandleFunc("POST /api/v1/auth/password", auth.changePassword)
	mux.HandleFunc("GET /api/v1/auth/me", auth.me)

	handler := jobHandler{
		creator:    config.CreateJob,
		finder:     config.GetJob,
		reports:    config.GetReport,
		files:      config.ListFiles,
		downloads:  config.DownloadFile,
		archive:    config.DownloadArchive,
		editor:     config.SubmitEdits,
		previews:   config.Preview,
		maxUploads: config.MaxUploadBytes,
		admin:      config.Admin,
	}
	mux.HandleFunc("POST /api/v1/jobs/order-fill", handler.createOrderFill)
	mux.HandleFunc("POST /api/v1/jobs/north-merge", handler.createNorthMerge)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", handler.getJob)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/report", handler.getReport)
	mux.HandleFunc("POST /api/v1/jobs/{job_id}/edits", handler.submitEdits)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files", handler.listFiles)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/archive", handler.downloadArchive)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}", handler.downloadFile)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}/preview", handler.previewMeta)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}/preview/window", handler.previewWindow)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}/preview/find", handler.previewFind)

	admin := adminHandler{admin: config.Admin, reset: config.Reset, lister: config.ListJobs}
	mux.HandleFunc("GET /api/v1/jobs", admin.listJobs)
	mux.HandleFunc("GET /api/v1/companies", admin.listCompanies)
	mux.HandleFunc("POST /api/v1/companies", admin.createCompany)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/disable", admin.disableCompany)
	mux.HandleFunc("GET /api/v1/companies/{company_id}/users", admin.listUsers)
	mux.HandleFunc("POST /api/v1/companies/{company_id}/users", admin.createUser)
	mux.HandleFunc("POST /api/v1/users/{user_id}/disable", admin.disableUser)
	mux.HandleFunc("POST /api/v1/users/{user_id}/reset", admin.resetUser)
	mux.HandleFunc("GET /api/v1/audit", admin.listAudit)

	protected := gate{
		next:           mux,
		auth:           config.Auth,
		allowedOrigins: config.AllowedOrigins,
		loginLimiter:   config.LoginLimiter,
		createLimiter:  config.CreateLimiter,
	}
	return withCORS(protected, config.AllowedOrigins)
}

type healthHandler struct{}

func (h healthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type metricsHandler struct {
	metrics MetricsReader
}

func (h metricsHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.metrics.Snapshot())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, errorResponse{Code: code, Message: message})
}
