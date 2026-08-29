package httpapi

import (
	"net/http"
	"strings"
)

const (
	corsAllowedMethods = "GET, POST, OPTIONS"
	corsAllowedHeaders = "Content-Type"
	corsMaxAgeSeconds  = "600"
)

type corsMiddleware struct {
	next           http.Handler
	allowedOrigins []string
}

// ParseAllowedOrigins reads a comma separated origin list. An empty value or "*"
// allows any origin, which is what the local docker stack and vite dev server need.
func ParseAllowedOrigins(value string) []string {
	origins := make([]string, 0)
	for _, entry := range strings.Split(value, ",") {
		trimmed := strings.TrimRight(strings.TrimSpace(entry), "/")
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			return []string{"*"}
		}
		origins = append(origins, trimmed)
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	return corsMiddleware{next: next, allowedOrigins: allowedOrigins}
}

func (m corsMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if allowed, ok := m.resolveOrigin(origin); ok {
		header := w.Header()
		header.Set("Access-Control-Allow-Origin", allowed)
		header.Set("Access-Control-Allow-Methods", corsAllowedMethods)
		header.Set("Access-Control-Allow-Headers", corsAllowedHeaders)
		header.Set("Access-Control-Max-Age", corsMaxAgeSeconds)
		header.Add("Vary", "Origin")
	}
	if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	m.next.ServeHTTP(w, r)
}

func (m corsMiddleware) resolveOrigin(origin string) (string, bool) {
	if origin == "" {
		return "", false
	}
	for _, allowed := range m.allowedOrigins {
		if allowed == "*" {
			return "*", true
		}
		if strings.EqualFold(allowed, origin) {
			return origin, true
		}
	}
	return "", false
}
