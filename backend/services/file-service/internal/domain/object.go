package domain

import "strings"

type Object struct {
	ID          string
	Key         string
	Name        string
	ContentType string
	Size        int64
	Body        []byte
	CompanyID   string
}

func PublicLogoKey(key string) bool {
	const prefix = "companies/"
	const suffix = "/logo"
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
	return id != "" && !strings.ContainsAny(id, `/`)
}

func LogoCompanyID(key string) string {
	if !PublicLogoKey(key) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(key, "companies/"), "/logo")
}
