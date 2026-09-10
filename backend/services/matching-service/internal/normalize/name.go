package normalize

import (
	"regexp"
	"strings"
)

var (
	cyrillicAnWordPattern      = regexp.MustCompile(`\bан\b`)
	latinAngiopharmWordPattern = regexp.MustCompile(`\bangiopharm\b`)
)

func NormalizeName(value string) string {
	text := cyrillicAnWordPattern.ReplaceAllString(NormalizeHeader(value), " ")
	text = latinAngiopharmWordPattern.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}
