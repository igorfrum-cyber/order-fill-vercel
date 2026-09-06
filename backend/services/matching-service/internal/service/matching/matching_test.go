package matching_test

import (
	"math"
	"testing"

	"order-fill/backend/services/matching-service/internal/domain"
	"order-fill/backend/services/matching-service/internal/normalize"
	"order-fill/backend/services/matching-service/internal/service/matching"
)

func TestNormalizeArticleAndName(t *testing.T) {
	t.Parallel()
	if got := normalize.NormalizeArticle(" АВ-12 х ", normalize.ArticleOptions{}); got != "AB12X" {
		t.Fatalf("got %q", got)
	}
	if got := normalize.NormalizeName("АН Косметичка непромокаемая"); got != "ан косметичка непромокаемая" {
		t.Fatalf("got %q", got)
	}
}

func TestSimilarityMatchesBrowserEngine(t *testing.T) {
	t.Parallel()
	got := matching.Similarity("Косметичка непромокаемая", "АН Косметичка непромокаемая")
	if math.Abs(got-0.9411764705882353) > 1e-9 {
		t.Fatalf("got %v", got)
	}
}

func TestArticleExactAndAlias(t *testing.T) {
	t.Parallel()
	svc := matching.New()
	blank := []domain.Item{{ID: "b1", Article: "123", Name: "Cream 50 ml"}}
	source := []domain.Item{{ID: "s1", Article: "MT123", Name: "Cream 50 ml"}}
	got := svc.Match(blank, source, matching.Options{PrefixAliases: []string{"MT"}})
	if len(got) != 1 || got[0].SourceItemID != "s1" || got[0].Reasons.Article != "alias" {
		t.Fatalf("%+v", got)
	}
	exact := svc.Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Same"}},
		[]domain.Item{{ID: "s1", Article: "A1", Name: "Same"}},
		matching.Options{},
	)
	if exact[0].Reasons.Article != "exact" || exact[0].SourceItemID != "s1" || exact[0].Category != domain.CategoryToOrder {
		t.Fatalf("%+v", exact)
	}
}

func TestNameFallbackAndSuspiciousVolume(t *testing.T) {
	t.Parallel()
	svc := matching.New()
	nameOnly := svc.Match(
		[]domain.Item{{ID: "b1", Article: "ZZ", Name: "Косметичка непромокаемая"}},
		[]domain.Item{{ID: "s1", Article: "", Name: "АН Косметичка непромокаемая", Rounded: 3}},
		matching.Options{},
	)
	if nameOnly[0].Category != domain.CategoryNeedsDecision || nameOnly[0].Reasons.Source != "name" {
		t.Fatalf("%+v", nameOnly)
	}
	volume := svc.Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Cream 50 ml"}},
		[]domain.Item{{ID: "s1", Article: "A1", Name: "Cream 30 ml"}},
		matching.Options{},
	)
	if volume[0].Category != domain.CategoryCheckNameOrVolume {
		t.Fatalf("%+v", volume)
	}
}

func TestDuplicatesAndStableCandidateOrder(t *testing.T) {
	t.Parallel()
	svc := matching.New()
	blank := []domain.Item{{ID: "b1", Article: "A1", Name: "Cream 50 ml"}}
	source := []domain.Item{
		{ID: "s2", Article: "A1", Name: "Cream 30 ml"},
		{ID: "s1", Article: "A1", Name: "Cream 50 ml"},
	}
	got := svc.Match(blank, source, matching.Options{})
	if got[0].SourceItemID != "s1" || got[0].Reasons.Duplicates != "chosen_best" {
		t.Fatalf("%+v", got)
	}
	if got[0].CandidateIDs[0] != "s1" || got[0].CandidateIDs[1] != "s2" {
		t.Fatalf("candidates %v", got[0].CandidateIDs)
	}
	if dups := matching.DuplicateSourceArticles(source); len(dups) != 1 || dups[0] != "A1" {
		t.Fatalf("source dups %v", dups)
	}
	if dups := matching.DuplicateBlankArticles([]domain.Item{{Article: "A1"}, {Article: "A1"}}); len(dups) != 1 {
		t.Fatalf("blank dups %v", dups)
	}
}

func TestNoArticleSourceRowsStayUnmatchedWithoutNameHit(t *testing.T) {
	t.Parallel()
	svc := matching.New()
	got := svc.Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Unknown"}},
		[]domain.Item{{ID: "s1", Article: "", Name: "Something else"}},
		matching.Options{},
	)
	if got[0].Category != domain.CategoryNotInSource {
		t.Fatalf("%+v", got)
	}
}

func TestSmartModeNeedsDecisionWhenDuplicatesAreClose(t *testing.T) {
	t.Parallel()
	svc := matching.New()
	got := svc.Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Cream"}},
		[]domain.Item{
			{ID: "s1", Article: "A1", Name: "Cream"},
			{ID: "s2", Article: "A1", Name: "Cream"},
		},
		matching.Options{Mode: domain.ModeSmart},
	)
	if got[0].Category != domain.CategoryNeedsDecision {
		t.Fatalf("%+v", got)
	}
}

func TestMergeChzJoinsCloneToBaseRow(t *testing.T) {
	t.Parallel()
	got := matching.New().MergeChz([]domain.Item{
		{ID: "5", Article: "AA04", Name: "АН Сыворотка 30 мл"},
		{ID: "6", Article: "AA04", Name: "ЧЗ АН Сыворотка 30 мл"},
		{ID: "7", Article: "BB01", Name: "АН Крем 50 мл"},
	}, matching.Options{})
	if len(got) != 1 || got[0].TargetID != "5" || len(got[0].CloneIDs) != 1 || got[0].CloneIDs[0] != "6" || got[0].NeedsDecision {
		t.Fatalf("%+v", got)
	}
}

func TestMergeChzSmartFlagsAmbiguousBaseRows(t *testing.T) {
	t.Parallel()
	got := matching.New().MergeChz([]domain.Item{
		{ID: "n1", Article: "A1", Name: "Сыворотка салициловая 30 мл"},
		{ID: "n2", Article: "A1", Name: "Сыворотка гиалуроновая 30 мл"},
		{ID: "c1", Article: "A1", Name: "ЧЗ Сыворотка 30 мл"},
	}, matching.Options{Mode: domain.ModeSmart})
	if len(got) != 1 || !got[0].NeedsDecision || got[0].CloneIDs[0] != "c1" {
		t.Fatalf("%+v", got)
	}
}

func TestFormConflictSetsCheckNameOrVolume(t *testing.T) {
	t.Parallel()
	got := matching.New().Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Крем 50 мл"}},
		[]domain.Item{{ID: "s1", Article: "A1", Name: "Сыворотка 50 мл"}},
		matching.Options{},
	)
	if got[0].Category != domain.CategoryCheckNameOrVolume || got[0].Reasons.Form != "conflict" {
		t.Fatalf("%+v", got)
	}
}

func TestSmartModeExplainsArticleVolumeFormAndDuplicateReasons(t *testing.T) {
	t.Parallel()
	svc := matching.New()
	got := svc.Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Крем 50 мл"}},
		[]domain.Item{{ID: "s1", Article: "A1", Name: "Сыворотка 30 мл"}},
		matching.Options{Mode: domain.ModeSmart},
	)
	if got[0].Category != domain.CategoryCheckNameOrVolume {
		t.Fatalf("%+v", got[0])
	}
	if got[0].Reasons.Article != "exact" || got[0].Reasons.Volume != "conflict" || got[0].Reasons.Form != "conflict" {
		t.Fatalf("%+v", got[0].Reasons)
	}
}

func TestSmartModeKeepsSingleArticleWarningNonBlocking(t *testing.T) {
	t.Parallel()
	got := matching.New().Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Крем"}},
		[]domain.Item{{ID: "s1", Article: "A1", Name: "Крем ночной"}},
		matching.Options{Mode: domain.ModeSmart},
	)
	if got[0].Category == domain.CategoryNeedsDecision {
		t.Fatalf("single article match with no conflict must not block: %+v", got[0])
	}
}
