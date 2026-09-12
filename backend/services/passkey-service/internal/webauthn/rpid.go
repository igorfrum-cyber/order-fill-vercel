package webauthn

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func resolveRPID(origin string, configured string) (string, error) {
	host, err := originHost(origin)
	if err != nil {
		return "", err
	}
	if ip := net.ParseIP(host); ip != nil {
		return "", fmt.Errorf("origin %s is an IP address; passkeys need a domain", origin)
	}
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return host, nil
	}
	if host == configured {
		return configured, nil
	}
	if strings.HasSuffix(host, "."+configured) {
		if isPublicSuffix(configured) {
			return host, nil
		}
		return configured, nil
	}
	return "", fmt.Errorf("origin %s does not match relying party %s", origin, configured)
}

func isPublicSuffix(value string) bool {
	return value == "localhost" || value == "local"
}

func originHost(origin string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid origin")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("invalid origin")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("invalid origin")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(host) {
		return "", fmt.Errorf("http origins are only allowed for local development")
	}
	return host, nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
