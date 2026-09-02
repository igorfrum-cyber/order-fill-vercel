package identity

import (
	"strconv"
	"strings"
	"unicode"
)

var cyrillic = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "i", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// LoginSlugFromName builds a URL slug from a company name.
func LoginSlugFromName(name, companyID string) string {
	slug := slugify(name)
	if slug == "" {
		return fallbackLoginSlug(companyID)
	}
	return slug
}

// UniqueLoginSlug returns a slug that is not reported as taken.
func UniqueLoginSlug(name, companyID string, taken func(string) bool) string {
	base := LoginSlugFromName(name, companyID)
	if taken == nil || !taken(base) {
		return base
	}
	candidate := base + "-" + shortCompanyID(companyID)
	if !taken(candidate) {
		return candidate
	}
	for n := 2; ; n++ {
		next := candidate + "-" + strconv.Itoa(n)
		if !taken(next) {
			return next
		}
	}
}

func fallbackLoginSlug(companyID string) string {
	return "company-" + shortCompanyID(companyID)
}

func shortCompanyID(companyID string) string {
	compact := strings.ToLower(strings.ReplaceAll(companyID, "-", ""))
	if compact == "" {
		return "x"
	}
	if len(compact) > 8 {
		return compact[:8]
	}
	return compact
}

func slugify(name string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if mapped, ok := cyrillic[r]; ok {
			if mapped == "" {
				continue
			}
			b.WriteString(mapped)
			prevHyphen = false
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '_' {
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
