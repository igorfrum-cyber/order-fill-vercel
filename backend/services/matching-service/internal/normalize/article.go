package normalize

import (
	"strings"
	"unicode"
)

type ArticleOptions struct {
	PreserveHyphen bool
}

var articleTranslation = map[rune]rune{
	'А': 'A', 'В': 'B', 'Е': 'E', 'К': 'K', 'М': 'M', 'Н': 'H', 'О': 'O',
	'Р': 'P', 'С': 'C', 'Т': 'T', 'Х': 'X', 'У': 'Y',
	'а': 'A', 'в': 'B', 'е': 'E', 'к': 'K', 'м': 'M', 'н': 'H', 'о': 'O',
	'р': 'P', 'с': 'C', 'т': 'T', 'х': 'X', 'у': 'Y',
}

func AsText(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
}

func NormalizeHeader(value string) string {
	text := strings.ReplaceAll(strings.ToLower(AsText(value)), "ё", "е")
	var b strings.Builder
	previousSpace := false
	for _, char := range text {
		allowed := unicode.IsLetter(char) || unicode.IsDigit(char) || char == '%'
		if allowed {
			b.WriteRune(char)
			previousSpace = false
			continue
		}
		if !previousSpace {
			b.WriteRune(' ')
			previousSpace = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func NormalizeArticle(value string, options ArticleOptions) string {
	var b strings.Builder
	for _, char := range AsText(value) {
		if translated, ok := articleTranslation[char]; ok {
			char = translated
		}
		char = unicode.ToUpper(char)
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || options.PreserveHyphen && char == '-' {
			b.WriteRune(char)
		}
	}
	return b.String()
}
