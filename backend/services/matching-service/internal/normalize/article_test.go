package normalize

import "testing"

func TestAsText(t *testing.T) {
	t.Parallel()

	if got := AsText(" \tA\nB \n"); got != "A B" {
		t.Fatalf("AsText() = %q, want %q", got, "A B")
	}
}

func TestNormalizeHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ё and case", in: " ЁЖИК КРЕМ ", want: "ежик крем"},
		{name: "punctuation collapses", in: "A---B / C", want: "a b c"},
		{name: "percent is retained", in: "Скидка 25%", want: "скидка 25%"},
		{name: "empty", in: " \n---\t", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeHeader(tt.in); got != tt.want {
				t.Fatalf("NormalizeHeader(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeArticle(t *testing.T) {
	t.Parallel()

	if got := NormalizeArticle(" АВ-12 х abc_9 ", ArticleOptions{}); got != "AB12XABC9" {
		t.Fatalf("NormalizeArticle() = %q, want %q", got, "AB12XABC9")
	}
	if got := NormalizeArticle("ab-12", ArticleOptions{PreserveHyphen: true}); got != "AB-12" {
		t.Fatalf("NormalizeArticle() with preserved hyphen = %q, want %q", got, "AB-12")
	}
}
