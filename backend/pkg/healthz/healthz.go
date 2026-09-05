package healthz

import (
	"context"
	"encoding/json"
	"net/http"
)

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func Live() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, Response{Status: "ok"})
	})
}

func Ready(check func(context.Context) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check == nil {
			write(w, http.StatusOK, Response{Status: "ok"})
			return
		}
		if err := check(r.Context()); err != nil {
			write(w, http.StatusServiceUnavailable, Response{Status: "not_ready", Error: err.Error()})
			return
		}
		write(w, http.StatusOK, Response{Status: "ok"})
	})
}

func write(w http.ResponseWriter, status int, body Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
