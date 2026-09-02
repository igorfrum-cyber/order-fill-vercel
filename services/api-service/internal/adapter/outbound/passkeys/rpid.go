package passkeys

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

const defaultDisplayName = "Order Fill"

func resolveRPID(origin string, configured string) (string, error) {
	host, err := originHost(origin)
	if err != nil {
		return "", err
	}
	configured = strings.TrimSpace(configured)
	if configured != "" {
		if host == configured || strings.HasSuffix(host, "."+configured) {
			return configured, nil
		}
		return "", fmt.Errorf("origin %s does not match relying party %s", origin, configured)
	}
	if ip := net.ParseIP(host); ip != nil {
		return host, nil
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "localhost", nil
	}
	return host, nil
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
	return host, nil
}
