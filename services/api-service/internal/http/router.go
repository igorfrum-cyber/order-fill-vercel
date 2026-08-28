package httpapi

import "net/http"

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthHandler{})
	return mux
}
