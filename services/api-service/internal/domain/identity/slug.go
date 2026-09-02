package identity

import (
	"fmt"
	"regexp"
	"strings"
)

var loginSlugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

var reservedLoginSlugs = map[string]struct{}{
	"admin": {}, "api": {}, "app": {}, "assets": {}, "c": {},
	"ftp": {}, "healthz": {}, "invite": {}, "localhost": {}, "login": {},
	"mail": {}, "metrics": {}, "public": {}, "static": {}, "www": {},
}

// ParseLoginSlug normalizes a public company login address.
func ParseLoginSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if slug == "" {
		return "", fmt.Errorf("%w: login slug is required", ErrInvalidLoginSlug)
	}
	if _, reserved := reservedLoginSlugs[slug]; reserved {
		return "", fmt.Errorf("%w: login slug is reserved", ErrInvalidLoginSlug)
	}
	if !loginSlugPattern.MatchString(slug) {
		return "", fmt.Errorf("%w: login slug must be latin letters, digits and hyphen", ErrInvalidLoginSlug)
	}
	return slug, nil
}
