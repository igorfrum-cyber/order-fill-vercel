package calculation

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

var (
	reBiopro     = regexp.MustCompile(`\bbiopro\b`)
	reBioPro     = regexp.MustCompile(`\bbio\s*pro\b`)
	reAN         = regexp.MustCompile(`\bан\b`)
	reAngio      = regexp.MustCompile(`\bangiopharm\b`)
	reNovacutan  = regexp.MustCompile(`\bновакутан\b`)
	reWordSpaces = regexp.MustCompile(`\s+`)
)

// NormalizeHeader matches origin/main workbookProcessor.normalizeHeader.
func NormalizeHeader(value string) string {
	s := strings.ToLower(strings.ReplaceAll(value, "ё", "е"))
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '%' {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// NormalizeName matches origin/main workbookProcessor.normalizeName.
func NormalizeName(value string) string {
	text := reAngio.ReplaceAllString(reAN.ReplaceAllString(NormalizeHeader(value), " "), " ")
	return strings.TrimSpace(reWordSpaces.ReplaceAllString(text, " "))
}

// NovacutanMatchKey matches origin/main workbookProcessor.novacutanMatchKey.
func NovacutanMatchKey(value string) string {
	text := reBioPro.ReplaceAllString(reBiopro.ReplaceAllString(NormalizeHeader(value), "bio pro"), "bio pro")
	if text == "" || strings.Contains(text, "термопакет") || strings.Contains(text, "хладоэлемент") {
		return ""
	}
	hasFillerMask := strings.Contains(text, "filler") || strings.Contains(text, "филлер") || strings.Contains(text, "маск") || strings.Contains(text, "mask")
	switch {
	case hasFillerMask && (strings.Contains(text, "глаз") || strings.Contains(text, "eye")):
		return "mask-eye"
	case hasFillerMask && (strings.Contains(text, "лица") || strings.Contains(text, "face")):
		return "mask-face"
	case strings.Contains(text, "sbio"):
		return "sbio"
	case strings.Contains(text, "ybio"):
		return "ybio"
	case strings.Contains(text, "bio pro"):
		return "bio-pro"
	case strings.Contains(text, "prima"):
		return "prima"
	case strings.Contains(text, "master"):
		return "master"
	case strings.Contains(text, "bright") || strings.Contains(text, "брайт"):
		return "bright"
	case strings.Contains(text, "gentle") || strings.Contains(text, "джентл") || strings.Contains(text, "джентел"):
		return "gentle"
	case strings.Contains(text, "fbio") && strings.Contains(text, "dvs") && strings.Contains(text, "light"):
		return "dvs-fbio-light"
	case strings.Contains(text, "fbio") && strings.Contains(text, "dvs") && strings.Contains(text, "medium"):
		return "dvs-fbio-medium"
	case strings.Contains(text, "fbio") && strings.Contains(text, "dvs") && strings.Contains(text, "volume"):
		return "dvs-fbio-volume"
	case strings.Contains(text, "fbio") && strings.Contains(text, "light"):
		return "fbio-light"
	case strings.Contains(text, "fbio") && strings.Contains(text, "medium"):
		return "fbio-medium"
	case strings.Contains(text, "fbio") && strings.Contains(text, "volume"):
		return "fbio-volume"
	case strings.Contains(text, "eye"):
		return "eye"
	default:
		return ""
	}
}

// NovacutanPositionKey matches origin/main workbookProcessor.novacutanPositionKey.
func NovacutanPositionKey(name string) string {
	matched := NovacutanMatchKey(name)
	if matched == "" {
		matched = reNovacutan.ReplaceAllString(NormalizeName(name), "novacutan")
	}
	return "novacutan:" + matched
}

// NovacutanMinimumQuantity matches origin/main workbookProcessor.novacutanMinimumQuantity.
func NovacutanMinimumQuantity(name string) float64 {
	text := NormalizeHeader(name)
	if (strings.Contains(text, "mask") || strings.Contains(text, "маск")) && (strings.Contains(text, "filler") || strings.Contains(text, "филлер")) {
		return 10
	}
	if strings.Contains(text, "fbio") || strings.Contains(text, "bright") || strings.Contains(text, "брайт") ||
		strings.Contains(text, "gentle") || strings.Contains(text, "джентл") || strings.Contains(text, "джентел") {
		return 50
	}
	return 100
}

// NovacutanSupplierUnitSize matches origin/main workbookProcessor.novacutanSupplierUnitSize.
func NovacutanSupplierUnitSize(name string) float64 {
	key := NovacutanMatchKey(name)
	if key == "mask-eye" || key == "mask-face" {
		return 5
	}
	return 1
}

func roundSupplierPack10(value float64) float64 {
	if value <= 0 {
		return 0
	}
	return math.Round(value/10) * 10
}
