package httpapi

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	msgAuthRequired  = "Нужно войти заново."
	msgNotFound      = "Не нашли то, что искали."
	msgJobGone       = "Эту выгрузку уже нельзя открыть. Обновите список."
	msgFileGone      = "Файл уже недоступен. Обновите страницу."
	msgConflict      = "Такая запись уже есть."
	msgGeneric       = "Что-то пошло не так. Попробуйте ещё раз."
	msgExcelFile     = "Нужен файл Excel: .xlsx или .xlsm."
	msgLoginSlug     = "Адрес входа — латиницей, цифрами и дефисом."
	msgPassword      = "Пароль не подходит. Проверьте требования и попробуйте снова." // #nosec G101 -- user-facing error, not a secret
	msgCompanyName   = "Напишите название компании."
	msgLoginRequired = "Напишите логин."
	msgLogo          = "Логотип должен быть картинкой PNG, JPEG или WebP."
	msgWrongBlank    = "Этот файл не похож на нужный бланк. Проверьте, что таблица из 1С и бланк поставщика не перепутаны."
)

var publicErrorPatterns = []struct {
	pattern *regexp.Regexp
	message string
}{
	{regexp.MustCompile(`(?i)authentication is required|unauthorized$`), msgAuthRequired},
	{regexp.MustCompile(`(?i)job was not found|report was not found`), msgJobGone},
	{regexp.MustCompile(`(?i)file was not found|preview was not found`), msgFileGone},
	{regexp.MustCompile(`(?i)\bnot found\b`), msgNotFound},
	{regexp.MustCompile(`(?i)\.xlsx or \.xlsm`), msgExcelFile},
	{regexp.MustCompile(`(?i)login slug must be latin|invalid login slug`), msgLoginSlug},
	{regexp.MustCompile(`(?i)invalid password`), msgPassword},
	{regexp.MustCompile(`(?i)company name is required`), msgCompanyName},
	{regexp.MustCompile(`(?i)login is required`), msgLoginRequired},
	{regexp.MustCompile(`(?i)logo must be png jpeg or webp|invalid logo`), msgLogo},
	{regexp.MustCompile(`(?i)conflict$`), msgConflict},
	{regexp.MustCompile(`(?i)invalid workbook|article, name|orderedFact|blank_files|source_file`), msgWrongBlank},
	{regexp.MustCompile(`(?i)dial tcp|connection refused|sql:|pq:`), msgGeneric},
}

func publicErrorMessage(code string, message string) string {
	stripped := stripTechnicalPrefix(message)
	if looksHumanRussian(stripped) {
		return stripped
	}
	switch code {
	case "unauthorized":
		return msgAuthRequired
	case "conflict":
		return msgConflict
	}
	for _, item := range publicErrorPatterns {
		if item.pattern.MatchString(message) || item.pattern.MatchString(stripped) {
			return item.message
		}
	}
	return msgGeneric
}

func stripTechnicalPrefix(message string) string {
	message = strings.TrimSpace(message)
	for _, prefix := range []string{"invalid workbook: ", "invalid job: ", "invalid login slug: ", "invalid logo: "} {
		message = strings.TrimPrefix(message, prefix)
	}
	return strings.TrimSpace(message)
}

func looksHumanRussian(text string) bool {
	if text == "" {
		return false
	}
	hasCyrillic := false
	for _, r := range text {
		if unicode.Is(unicode.Cyrillic, r) {
			hasCyrillic = true
			break
		}
	}
	if !hasCyrillic {
		return false
	}
	lower := strings.ToLower(text)
	for _, leak := range []string{"article", "orderedfact", "intransit", "source_file", "blank_files", "invalid workbook"} {
		if strings.Contains(lower, leak) {
			return false
		}
	}
	return true
}
