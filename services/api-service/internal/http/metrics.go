package httpapi

import "net/http"

type MetricsReader interface {
	Snapshot() map[string]int64
}

type metricsHandler struct {
	metrics MetricsReader
}

func (h metricsHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.metrics.Snapshot())
}
