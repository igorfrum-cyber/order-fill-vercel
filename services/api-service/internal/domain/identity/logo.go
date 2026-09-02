package identity

import (
	"bytes"
	"fmt"
)

const LogoMaxBytes = 512 << 10

func CompanyLogoKey(companyID string) string {
	return "companies/" + companyID + "/logo"
}

func (c Company) HasLogo() bool {
	return c.LogoContentType != ""
}

// ParseLogo accepts PNG, JPEG and WebP up to LogoMaxBytes.
func ParseLogo(content []byte) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("%w: logo is required", ErrInvalidLogo)
	}
	if len(content) > LogoMaxBytes {
		return "", fmt.Errorf("%w: logo is too large", ErrInvalidLogo)
	}
	switch {
	case bytes.HasPrefix(content, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png", nil
	case len(content) >= 3 && content[0] == 0xff && content[1] == 0xd8 && content[2] == 0xff:
		return "image/jpeg", nil
	case len(content) >= 12 && bytes.Equal(content[0:4], []byte("RIFF")) && bytes.Equal(content[8:12], []byte("WEBP")):
		return "image/webp", nil
	default:
		return "", fmt.Errorf("%w: logo must be png jpeg or webp", ErrInvalidLogo)
	}
}
