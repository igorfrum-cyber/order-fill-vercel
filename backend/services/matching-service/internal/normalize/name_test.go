package normalize

import "testing"

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "keeps standalone cyrillic abbreviation", in: "АН Крем 50 мл", want: "ан крем 50 мл"},
		{name: "removes latin brand word", in: "Angiopharm cream", want: "cream"},
		{name: "normalizes whitespace and case", in: "  КРЕМ\n ночной  ", want: "крем ночной"},
		{name: "keeps abbreviation inside a word", in: "АНТИВОЗРАСТНОЙ крем", want: "антивозрастной крем"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeName(tt.in); got != tt.want {
				t.Fatalf("NormalizeName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
