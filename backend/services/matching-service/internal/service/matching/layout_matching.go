package matching

import (
	"order-fill/backend/services/matching-service/internal/domain"
	"order-fill/backend/services/matching-service/internal/normalize"
	"slices"
	"strings"
)

func chooseSothysFallback(items []domain.Item, name, unit string) (scored, bool) {
	ranked := []scored{}
	unit = normalize.NormalizeHeader(unit)
	for _, item := range items {
		if item.Article == "" && (unit == "" || strings.Contains(normalize.NormalizeHeader(item.Name), unit)) {
			ranked = append(ranked, scored{item: item, score: volumeAwareSimilarity(name, item.Name, unit)})
		}
	}
	slices.SortStableFunc(ranked, func(a, b scored) int {
		if a.score > b.score {
			return -1
		}
		if a.score < b.score {
			return 1
		}
		return 0
	})
	if len(ranked) == 0 {
		return scored{}, false
	}
	if ranked[0].score < .72 || len(ranked) > 1 && ranked[0].score-ranked[1].score < .08 {
		return scored{score: ranked[0].score}, false
	}
	return ranked[0], true
}
func novacutanKey(value string) string {
	text := normalize.NormalizeHeader(value)
	text = strings.ReplaceAll(text, "biopro", "bio pro")
	if text == "" || strings.Contains(text, "термопакет") || strings.Contains(text, "хладоэлемент") {
		return ""
	}
	mask := strings.Contains(text, "filler") || strings.Contains(text, "филлер") || strings.Contains(text, "маск") || strings.Contains(text, "mask")
	if mask && (strings.Contains(text, "глаз") || strings.Contains(text, "eye")) {
		return "mask-eye"
	}
	if mask && (strings.Contains(text, "лица") || strings.Contains(text, "face")) {
		return "mask-face"
	}
	for _, key := range []string{"sbio", "ybio", "bio pro", "prima", "master"} {
		if strings.Contains(text, key) {
			return strings.ReplaceAll(key, " ", "-")
		}
	}
	if strings.Contains(text, "bright") || strings.Contains(text, "брайт") {
		return "bright"
	}
	if strings.Contains(text, "gentle") || strings.Contains(text, "джентл") || strings.Contains(text, "джентел") {
		return "gentle"
	}
	if strings.Contains(text, "fbio") {
		for _, kind := range []string{"light", "medium", "volume"} {
			if strings.Contains(text, kind) {
				prefix := "fbio-"
				if strings.Contains(text, "dvs") {
					prefix = "dvs-" + prefix
				}
				return prefix + kind
			}
		}
	}
	if strings.Contains(text, "eye") {
		return "eye"
	}
	return ""
}
func resolveNovacutan(blank domain.Item, source []domain.Item) domain.Result {
	key := novacutanKey(blank.Name)
	pool := []domain.Item{}
	for _, item := range source {
		if key != "" && novacutanKey(item.Name) == key {
			pool = append(pool, item)
		}
	}
	selected := scored{}
	ok := false
	if len(pool) > 0 {
		beauty := func(s string) bool {
			text := normalize.NormalizeHeader(s)
			return strings.Contains(text, "бьюти") || strings.Contains(text, "beauty")
		}
		aligned := slices.DeleteFunc(slices.Clone(pool), func(i domain.Item) bool { return beauty(i.Name) != beauty(blank.Name) })
		if len(aligned) > 0 {
			pool = aligned
		}
		nonBonus := slices.DeleteFunc(slices.Clone(pool), func(i domain.Item) bool { return strings.Contains(normalize.NormalizeHeader(i.Name), "бонус") })
		if len(nonBonus) > 0 {
			pool = nonBonus
		}
		ranked := []scored{}
		for _, item := range pool {
			ranked = append(ranked, scored{item: item, score: max(.92, volumeAwareSimilarity(blank.Name, item.Name, blank.Volume))})
		}
		slices.SortStableFunc(ranked, func(a, b scored) int {
			ap, bp := a.item.Recommended > 0, b.item.Recommended > 0
			if ap != bp {
				if ap {
					return -1
				}
				return 1
			}
			if a.score > b.score {
				return -1
			}
			if a.score < b.score {
				return 1
			}
			return 0
		})
		selected, ok = ranked[0], true
	} else {
		selected, ok = chooseCandidate(source, blank.Name, blank.Volume)
	}
	if !ok || selected.score < .72 {
		return domain.Result{BlankItemID: blank.ID, Category: domain.CategoryNotInSource, Reasons: domain.Reasons{Source: reasonNone}}
	}
	category := domain.CategoryToOrder
	if selected.score < .85 {
		category = domain.CategoryCheckNameOrVolume
	}
	if selected.item.Rounded <= 0 {
		category = domain.CategoryOrderNotNeeded
	}
	return domain.Result{BlankItemID: blank.ID, SourceItemID: selected.item.ID, Category: category, Score: selected.score, Reasons: domain.Reasons{Source: reasonName, Name: reasonSimilar}, CandidateIDs: []string{selected.item.ID}}
}
