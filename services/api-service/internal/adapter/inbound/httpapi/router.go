// Package httpapi is the driving adapter: it translates HTTP into use case
// calls and domain results back into JSON.
package httpapi

import (
	"encoding/json"
	"net/http"
)

// MetricsReader exposes a counters snapshot for the /metrics endpoint.
type MetricsReader interface {
	Snapshot() map[string]int64
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
	Metrics         MetricsReader
	AllowedOrigins  []string
	MaxUploadBytes  int64
}

func NewRouter(config Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler{})
	if config.Metrics != nil {
		mux.Handle("GET /metrics", metricsHandler{metrics: config.Metrics})
	}

	handler := jobHandler{
		creator:    config.CreateJob,
		finder:     config.GetJob,
		reports:    config.GetReport,
		files:      config.ListFiles,
		downloads:  config.DownloadFile,
		archive:    config.DownloadArchive,
		editor:     config.SubmitEdits,
		maxUploads: config.MaxUploadBytes,
	}
	mux.HandleFunc("POST /api/v1/jobs/order-fill", handler.createOrderFill)
	mux.HandleFunc("POST /api/v1/jobs/north-merge", handler.createNorthMerge)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", handler.getJob)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/report", handler.getReport)
	mux.HandleFunc("POST /api/v1/jobs/{job_id}/edits", handler.submitEdits)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files", handler.listFiles)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/archive", handler.downloadArchive)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files/{file_id}", handler.downloadFile)

	return withCORS(mux, config.AllowedOrigins)
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
