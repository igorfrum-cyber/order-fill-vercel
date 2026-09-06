package normalize

import "testing"

func TestAsText(t *testing.T) {
	t.Parallel()
	if got := AsText(nil); got != "" {
		t.Fatalf("nil = %q, want empty", got)
	}
	if got := AsText([]byte("  hi\nthere ")); got != "hi there" {
		t.Fatalf("bytes = %q", got)
	}
	if got := AsText(12); got != "12" {
		t.Fatalf("number = %q", got)
	}
}

func TestNormalizeCategory(t *testing.T) {
	t.Parallel()
	if got := NormalizeCategory(" а+ "); got != "A+" {
		t.Fatalf("got %q, want A+", got)
	}
}

func TestRoundHalfUp(t *testing.T) {
	t.Parallel()
	if got := RoundHalfUp(1.5); got != 2 {
		t.Fatalf("1.5 → %d, want 2", got)
	}
	if got := RoundHalfUp(1.4); got != 1 {
		t.Fatalf("1.4 → %d, want 1", got)
	}
}

func TestNormalizeArticleTransliteratesLookalikeCyrillic(t *testing.T) {
	if got := NormalizeArticle(" АВ-12 х ", ArticleOptions{}); got != "AB12X" {
		t.Fatalf("expected AB12X, got %q", got)
	}
	if got := NormalizeArticle(" АВ-12 х ", ArticleOptions{PreserveHyphen: true}); got != "AB-12X" {
		t.Fatalf("expected AB-12X, got %q", got)
	}
}

func TestNormalizeNameKeepsCyrillicAnLikeTheBrowserEngine(t *testing.T) {
	// workbookProcessor.js uses ASCII \b, which does not treat "ан" as a word.
	// Stripping it as a token made "АН Крем" look identical to "Крем" and
	// inflated every match percentage against the original site.
	if got := NormalizeName("АН Косметичка непромокаемая"); got != "ан косметичка непромокаемая" {
		t.Fatalf("cyrillic brand prefix must stay in the comparable name, got %q", got)
	}
	if got := NormalizeName("Angiopharm cream 50 ml"); got != "cream 50 ml" {
		t.Fatalf("latin brand word must still be stripped, got %q", got)
	}
}

func TestParseNumberExtractsCommaDecimal(t *testing.T) {
	t.Parallel()
	got, ok := ParseNumber(" 1 234,50 шт")
	if !ok || got != 1234.5 {
		t.Fatalf("expected 1234.5, got %v ok=%v", got, ok)
	}
	if _, ok := ParseNumber(""); ok {
		t.Fatal("empty must not parse")
	}
	if _, ok := ParseNumber("нет числа"); ok {
		t.Fatal("text without digits must not parse")
	}
}
