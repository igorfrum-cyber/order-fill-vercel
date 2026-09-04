package httpapi

import "testing"

func TestPublicErrorMessageUsesRussian(t *testing.T) {
	cases := []struct {
		code    string
		message string
		want    string
	}{
		{"unauthorized", "authentication is required", "Нужно войти заново."},
		{"not_found", "job was not found", "Эту выгрузку уже нельзя открыть. Обновите список."},
		{"not_found", "not found", "Не нашли то, что искали."},
		{"conflict", "conflict", "Такая запись уже есть."},
		{"bad_request", "invalid job: file \"a.pdf\" must be .xlsx or .xlsm", "Нужен файл Excel: .xlsx или .xlsm."},
		{"bad_request", "invalid login slug: login slug must be latin letters, digits and hyphen", "Адрес входа — латиницей, цифрами и дефисом."},
		{"bad_request", "invalid password", "Пароль не подходит. Проверьте требования и попробуйте снова."},
		{"create_job_failed", "dial tcp 127.0.0.1:5432: connect: connection refused", "Что-то пошло не так. Попробуйте ещё раз."},
	}
	for _, tc := range cases {
		got := publicErrorMessage(tc.code, tc.message)
		if got != tc.want {
			t.Fatalf("%s %q: got %q, want %q", tc.code, tc.message, got, tc.want)
		}
	}
}
