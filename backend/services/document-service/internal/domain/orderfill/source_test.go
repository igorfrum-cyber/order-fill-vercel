package orderfill

import (
	"testing"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
)

func TestArticleKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		article  string
		prefixes []string
		want     []string
	}{
		{name: "empty", article: ""},
		{name: "no aliases", article: "123", want: []string{"123"}},
		{name: "digits gain prefix", article: "123", prefixes: []string{"MT"}, want: []string{"123", "MT123"}},
		{name: "prefixed loses prefix", article: "MT123", prefixes: []string{"MT"}, want: []string{"MT123", "123"}},
		{name: "prefix without trailing digits stays", article: "MTABC", prefixes: []string{"MT"}, want: []string{"MTABC"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := articleKeys(tt.article, brand.RuleConfig{ArticlePrefixAliases: tt.prefixes})
			if len(got) != len(tt.want) {
				t.Fatalf("keys = %v, want %v", got, tt.want)
			}
			for i, key := range tt.want {
				if got[i] != key {
					t.Fatalf("keys = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCandidatesForMatchesLevissimePrefixBothWays(t *testing.T) {
	t.Parallel()
	rule := brand.Rule("levissime")
	tests := []struct {
		name          string
		sourceArticle string
		blankArticle  string
	}{
		{name: "query digits", sourceArticle: "MT123", blankArticle: "123"},
		{name: "query prefixed", sourceArticle: "123", blankArticle: "MT123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := BuildSourceContext(Source{Items: []SourceItem{{
				RowIndex: 1,
				Article:  tt.sourceArticle,
				Name:     "Cream",
			}}}, rule)
			got := ctx.CandidatesFor(tt.blankArticle, rule)
			if len(got) != 1 {
				t.Fatalf("candidates = %d, want 1", len(got))
			}
			if got[0].Article != tt.sourceArticle {
				t.Fatalf("article = %q, want %q", got[0].Article, tt.sourceArticle)
			}
		})
	}
}

func TestCandidatesForKeepsSothysHyphen(t *testing.T) {
	t.Parallel()
	rule := brand.Rule("sothys")
	opts := brand.ArticleNormalizeOptions(rule)
	hyphenated := normalize.NormalizeArticle("110-15", opts)
	stripped := normalize.NormalizeArticle("11015", opts)
	if hyphenated != "110-15" {
		t.Fatalf("sothys article = %q, want 110-15", hyphenated)
	}
	ctx := BuildSourceContext(Source{Items: []SourceItem{{
		RowIndex: 1,
		Article:  hyphenated,
		Name:     "Serum",
	}}}, rule)
	if got := ctx.CandidatesFor(hyphenated, rule); len(got) != 1 {
		t.Fatalf("hyphenated lookup candidates = %d, want 1", len(got))
	}
	if got := ctx.CandidatesFor(stripped, rule); len(got) != 0 {
		t.Fatalf("stripped hyphen %q must not match sothys %q", stripped, hyphenated)
	}
}

func TestReadSourceExtractsRows(t *testing.T) {
	t.Parallel()
	source, err := ReadSource(newFakeWorkbook("Заказ", sourceGrid()), "2026-09", brand.Rule("angiopharm"))
	if err != nil {
		t.Fatalf("ReadSource: %v", err)
	}
	if len(source.Items) != 4 {
		t.Fatalf("items = %d, want 4", len(source.Items))
	}
}
