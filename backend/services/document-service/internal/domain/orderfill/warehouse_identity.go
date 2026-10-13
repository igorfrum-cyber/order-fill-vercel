package orderfill

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
)

var locationChzPrefix = regexp.MustCompile(`(?i)^[\s\p{Z}\x{000B}\x{FEFF}]*чз[\s\p{Z}\x{000B}\x{FEFF}]*\+?[\s\p{Z}\x{000B}\x{FEFF}]*`)
var locationLatinToken = regexp.MustCompile(`(?i)[a-z][a-z0-9]*`)
var locationRussian = regexp.MustCompile(`[а-яё]`)
var locationEnglishSegment = regexp.MustCompile(`^[A-Za-z0-9\s\p{Z}\x{000B}\x{FEFF}+().,-]+$`)
var locationEnglishProduct = regexp.MustCompile(`^[A-Za-z\s\p{Z}\x{000B}\x{FEFF}+().-]+$`)
var locationVolume = regexp.MustCompile(`\d+(?:[.,]\d+)?[\s\p{Z}\x{000B}\x{FEFF}]*(?:мл|мг|г|л)`)

func stripLocationChz(value string) string { return locationChzPrefix.ReplaceAllString(value, "") }
func hasChzPrefix(value string) bool       { return locationChzPrefix.MatchString(value) }
func locationCanonicalArticle(article string, rule brand.RuleConfig) string {
	for _, prefix := range rule.ArticlePrefixAliases {
		if rest, ok := strings.CutPrefix(article, prefix); ok && digitsPattern.MatchString(rest) {
			return rest
		}
	}
	return article
}
func locationProductKey(item *locationRow, rule brand.RuleConfig) string {
	if item.article != "" {
		return "article:" + locationCanonicalArticle(item.article, rule)
	}
	if rule.Key == "novacutan" {
		return "name:novacutan:" + locationNovacutanKey(item.name)
	}
	return "name:" + cleanLocationName(item.name)
}
func locationComparableName(value string, rule brand.RuleConfig) string {
	name := normalize.NormalizeHeader(value)
	name = strings.TrimPrefix(name, "чз ")
	name = normalize.NormalizeName(name)
	if rule.Key == "levissime" {
		name = strings.TrimPrefix(strings.TrimPrefix(name, "levissime "), "lev ")
		name = strings.ReplaceAll(name, "отбеливающая", "осветляющая")
	}
	return name
}
func locationRussianIdentity(value string, rule brand.RuleConfig) string {
	parts := strings.Split(stripLocationChz(value), "/")
	if len(parts) > 2 || len(parts) == 2 && !locationEnglishSegment.MatchString(parts[1]) {
		return ""
	}
	if !locationRussian.MatchString(strings.ToLower(parts[0])) || len(locationVolumeIndices(parts[0])) == 0 {
		return ""
	}
	return strings.Join(strings.Fields(locationLatinToken.ReplaceAllString(locationComparableName(parts[0], rule), " ")), " ")
}
func locationEnglishIdentity(value string) string {
	parts := strings.Split(value, "/")
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" || part[0] < 'A' || part[0] > 'Z' && part[0] < 'a' || part[0] > 'z' {
			continue
		}
		if loc := locationVolumeIndices(part); len(loc) > 0 {
			part = strings.TrimSpace(part[:loc[0][0]])
		}
		if !locationEnglishProduct.MatchString(part) {
			return ""
		}
		name := normalize.NormalizeHeader(part)
		if len(strings.Fields(name)) < 3 {
			return ""
		}
		return name
	}
	return ""
}
func locationVolumeSignature(value string) string {
	values := []string{}
	for _, index := range locationVolumeMatches(strings.ToLower(value)) {
		volume := strings.ReplaceAll(strings.Join(strings.Fields(strings.ToLower(value)[index[0]:index[1]]), ""), ",", ".")
		values = append(values, volume)
	}
	slices.Sort(values)
	return strings.Join(slices.Compact(values), ";")
}

func compatibleLocationProducts(left, right *locationRow, rule brand.RuleConfig) bool {
	if rule.Key == "levissime" && left.article != "" && right.article != "" && locationCanonicalArticle(left.article, rule) == locationCanonicalArticle(right.article, rule) {
		identity := locationRussianIdentity(left.name, rule)
		if identity != "" && identity == locationRussianIdentity(right.name, rule) {
			return true
		}
		english := locationEnglishIdentity(left.name)
		volume := locationVolumeSignature(left.name)
		if english != "" && volume != "" && english == locationEnglishIdentity(right.name) && volume == locationVolumeSignature(right.name) {
			return true
		}
		return locationComparableName(left.name, rule) == locationComparableName(right.name, rule)
	}
	return compatibleLocationNames(left.name, right.name)
}

func locationNovacutanKey(value string) string {
	text := normalize.NormalizeHeader(value)
	text = regexp.MustCompile(`\bbiopro\b`).ReplaceAllString(text, "bio pro")
	if !strings.Contains(text, "термопакет") && !strings.Contains(text, "хладоэлемент") {
		mask := strings.Contains(text, "filler") || strings.Contains(text, "филлер") || strings.Contains(text, "маск") || strings.Contains(text, "mask")
		if mask && (strings.Contains(text, "глаз") || strings.Contains(text, "eye")) {
			return "mask-eye"
		}
		if mask && (strings.Contains(text, "лица") || strings.Contains(text, "face")) {
			return "mask-face"
		}
		for _, key := range []string{"sbio", "ybio", "bio pro", "prima", "master"} {
			if strings.Contains(text, key) {
				return strings.ReplaceAll(key, " ", "-")
			}
		}
		if strings.Contains(text, "bright") || strings.Contains(text, "брайт") {
			return "bright"
		}
		if strings.Contains(text, "gentle") || strings.Contains(text, "джентл") || strings.Contains(text, "джентел") {
			return "gentle"
		}
		if strings.Contains(text, "fbio") {
			for _, kind := range []string{"light", "medium", "volume"} {
				if strings.Contains(text, kind) {
					prefix := "fbio-"
					if strings.Contains(text, "dvs") {
						prefix = "dvs-" + prefix
					}
					return prefix + kind
				}
			}
		}
		if strings.Contains(text, "eye") {
			return "eye"
		}
	}
	return normalize.NormalizeName(value)
}

func locationVolumeIndices(value string) [][]int {
	text := strings.ToLower(value)
	out := [][]int{}
	for _, loc := range locationVolume.FindAllStringIndex(text, -1) {
		if loc[1] < len(text) {
			r, _ := utf8.DecodeRuneInString(text[loc[1]:])
			if r >= 'а' && r <= 'я' || r == 'ё' {
				continue
			}
		}
		out = append(out, loc)
	}
	return out
}

func locationVolumeMatches(value string) [][]int {
	matches := [][]int{}
	for _, index := range locationVolume.FindAllStringIndex(value, -1) {
		if index[1] < len(value) {
			next, _ := utf8.DecodeRuneInString(value[index[1]:])
			if next >= 'а' && next <= 'я' || next == 'ё' {
				continue
			}
		}
		matches = append(matches, index)
	}
	return matches
}
