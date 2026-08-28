package domain

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
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

var numberPattern = regexp.MustCompile(`-?\d+(?:\.\d+)?`)

func AsText(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(toString(value), "\n", " "))
}

func NormalizeHeader(value any) string {
	text := strings.ReplaceAll(strings.ToLower(AsText(value)), "ё", "е")
	var builder strings.Builder
	previousSpace := false
	for _, char := range text {
		allowed := unicode.IsLetter(char) || unicode.IsDigit(char) || char == '%'
		if allowed {
			builder.WriteRune(char)
			previousSpace = false
			continue
		}
		if !previousSpace {
			builder.WriteRune(' ')
			previousSpace = true
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func NormalizeArticle(value any, options ArticleOptions) string {
	var builder strings.Builder
	for _, char := range AsText(value) {
		if translated, ok := articleTranslation[char]; ok {
			char = translated
		}
		char = unicode.ToUpper(char)
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || options.PreserveHyphen && char == '-' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func NormalizeName(value any) string {
	words := strings.Fields(NormalizeHeader(value))
	filtered := words[:0]
	for _, word := range words {
		if word == "ан" || word == "angiopharm" {
			continue
		}
		filtered = append(filtered, word)
	}
	return strings.Join(filtered, " ")
}

func NormalizeCategory(value any) string {
	var builder strings.Builder
	for _, char := range AsText(value) {
		if translated, ok := articleTranslation[char]; ok {
			char = translated
		}
		char = unicode.ToUpper(char)
		if !unicode.IsSpace(char) {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func ParseNumber(value any) (float64, bool) {
	text := AsText(value)
	if text == "" {
		return 0, false
	}
	normalized := strings.ReplaceAll(strings.Join(strings.Fields(text), ""), ",", ".")
	if number, err := strconv.ParseFloat(normalized, 64); err == nil {
		return number, true
	}
	match := numberPattern.FindString(normalized)
	if match == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, false
	}
	return number, true
}

func RoundHalfUp(value float64) int {
	return int(math.Floor(value + 0.5))
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
