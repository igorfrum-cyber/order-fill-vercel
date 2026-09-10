package httpapi

import (
	"bytes"
	"fmt"
)

const logoMaxBytes = 512 << 10

func parseLogo(content []byte) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("logo is required")
	}
	if len(content) > logoMaxBytes {
		return "", fmt.Errorf("logo is too large")
	}
	switch {
	case bytes.HasPrefix(content, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png", nil
	case len(content) >= 3 && content[0] == 0xff && content[1] == 0xd8 && content[2] == 0xff:
		return "image/jpeg", nil
	case len(content) >= 12 && bytes.Equal(content[0:4], []byte("RIFF")) && bytes.Equal(content[8:12], []byte("WEBP")):
		return "image/webp", nil
	default:
		return "", fmt.Errorf("logo must be png jpeg or webp")
	}
}

func companyLogoKey(companyID string) string {
	return "companies/" + companyID + "/logo"
}

func canManageCompany(user User, companyID string) bool {
	if user.Role == "platform_admin" {
		return true
	}
	return (user.Role == "company_owner" || user.Role == "company_admin") && user.CompanyID != "" && user.CompanyID == companyID
}
