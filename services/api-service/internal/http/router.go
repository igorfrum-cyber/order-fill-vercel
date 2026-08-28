package httpapi

import "net/http"

type RouterOption func(*routerConfig)

type routerConfig struct {
	jobCreator    JobCreator
	jobReader     JobReader
	jobReporter   JobReporter
	jobEditor     JobEditor
	jobFileLister JobFileLister
	metrics       MetricsReader
}

func WithJobCreator(creator JobCreator) RouterOption {
	return func(config *routerConfig) {
		config.jobCreator = creator
	}
}

func WithJobReader(reader JobReader) RouterOption {
	return func(config *routerConfig) {
		config.jobReader = reader
	}
}

func WithJobService(service interface {
	JobCreator
	JobReader
	JobReporter
	JobEditor
	JobFileLister
}) RouterOption {
	return func(config *routerConfig) {
		config.jobCreator = service
		config.jobReader = service
		config.jobReporter = service
		config.jobEditor = service
		config.jobFileLister = service
	}
}

func WithMetrics(metrics MetricsReader) RouterOption {
	return func(config *routerConfig) {
		config.metrics = metrics
	}
}

func NewRouter(options ...RouterOption) http.Handler {
	config := routerConfig{}
	for _, option := range options {
		option(&config)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler{})
	if config.metrics != nil {
		mux.Handle("GET /metrics", metricsHandler{metrics: config.metrics})
	}
	jobHandler := jobHandler{
		creator:    config.jobCreator,
		reader:     config.jobReader,
		reporter:   config.jobReporter,
		editor:     config.jobEditor,
		fileLister: config.jobFileLister,
	}
	mux.HandleFunc("POST /api/v1/jobs/order-fill", jobHandler.createOrderFill)
	mux.HandleFunc("POST /api/v1/jobs/north-merge", jobHandler.createNorthMerge)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", jobHandler.getJob)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/report", jobHandler.getReport)
	mux.HandleFunc("POST /api/v1/jobs/{job_id}/edits", jobHandler.submitEdits)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}/files", jobHandler.listFiles)
	return mux
}
