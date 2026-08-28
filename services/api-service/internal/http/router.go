package httpapi

import "net/http"

type RouterOption func(*routerConfig)

type routerConfig struct {
	jobCreator JobCreator
	jobReader  JobReader
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
}) RouterOption {
	return func(config *routerConfig) {
		config.jobCreator = service
		config.jobReader = service
	}
}

func NewRouter(options ...RouterOption) http.Handler {
	config := routerConfig{}
	for _, option := range options {
		option(&config)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler{})
	jobHandler := jobHandler{creator: config.jobCreator, reader: config.jobReader}
	mux.HandleFunc("POST /api/v1/jobs/order-fill", jobHandler.createOrderFill)
	mux.HandleFunc("POST /api/v1/jobs/north-merge", jobHandler.createNorthMerge)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", jobHandler.getJob)
	return mux
}
