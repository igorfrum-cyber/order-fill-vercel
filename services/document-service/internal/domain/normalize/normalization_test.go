package normalize

import "testing"

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
	got, ok := ParseNumber(" 1 234,50 шт")
	if !ok || got != 1234.5 {
		t.Fatalf("expected 1234.5, got %v ok=%v", got, ok)
	}
}
