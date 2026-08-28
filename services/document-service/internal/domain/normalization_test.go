package domain

import "testing"

func TestNormalizeArticleTransliteratesLookalikeCyrillic(t *testing.T) {
	if got := NormalizeArticle(" АВ-12 х ", ArticleOptions{}); got != "AB12X" {
		t.Fatalf("expected AB12X, got %q", got)
	}
	if got := NormalizeArticle(" АВ-12 х ", ArticleOptions{PreserveHyphen: true}); got != "AB-12X" {
		t.Fatalf("expected AB-12X, got %q", got)
	}
}

func TestParseNumberExtractsCommaDecimal(t *testing.T) {
	got, ok := ParseNumber(" 1 234,50 шт")
	if !ok || got != 1234.5 {
		t.Fatalf("expected 1234.5, got %v ok=%v", got, ok)
	}
}
