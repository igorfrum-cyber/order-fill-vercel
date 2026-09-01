package httpapi

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func csrfAllowed(r *http.Request, allowedOrigins []string) bool {
	if r.Header.Get("X-Requested-With") != "fetch" {
		return false
	}
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin == "" {
		return true
	}
	return originAllowed(origin, allowedOrigins)
}

func originAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if originsMatch(strings.TrimRight(allowed, "/"), origin) {
			return true
		}
	}
	return false
}

func originsMatch(allowed string, origin string) bool {
	if strings.EqualFold(allowed, origin) {
		return true
	}
	allowedURL, err := url.Parse(allowed)
	if err != nil || allowedURL.Scheme == "" || allowedURL.Host == "" {
		return false
	}
	originURL, err := url.Parse(origin)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" {
		return false
	}
	if !strings.EqualFold(allowedURL.Scheme, originURL.Scheme) {
		return false
	}
	if originPort(allowedURL) != originPort(originURL) {
		return false
	}
	return isLoopbackHost(allowedURL.Hostname()) && isLoopbackHost(originURL.Hostname())
}

func originPort(parsed *url.URL) string {
	if port := parsed.Port(); port != "" {
		return port
	}
	if parsed.Scheme == "https" {
		return "443"
	}
	return "80"
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
