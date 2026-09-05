package matching

import (
	"slices"

	"order-fill/backend/services/matching-service/internal/domain"
)

func DuplicateBlankArticles(items []domain.Item) []string {
	counts := map[string]int{}
	for _, item := range items {
		if item.Article == "" {
			continue
		}
		counts[item.Article]++
	}
	out := make([]string, 0)
	for article, n := range counts {
		if n > 1 {
			out = append(out, article)
		}
	}
	slices.Sort(out)
	return out
}

func DuplicateSourceArticles(items []domain.Item) []string {
	return DuplicateBlankArticles(items)
}
