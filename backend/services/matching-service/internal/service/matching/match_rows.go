package matching

import (
	"regexp"
	"slices"
	"strings"

	"order-fill/backend/services/matching-service/internal/domain"
	"order-fill/backend/services/matching-service/internal/normalize"
)

const nameMatchThreshold = 0.32

var digitsPattern = regexp.MustCompile(`^\d+$`)

type Options struct {
	Mode           domain.Mode
	PrefixAliases  []string
	PreserveHyphen bool
}

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) Match(blank, source []domain.Item, opts Options) []domain.Result {
	index := indexSource(source, opts)
	out := make([]domain.Result, 0, len(blank))
	for _, item := range blank {
		out = append(out, resolve(item, index, opts))
	}
	return out
}

func (s *Service) NormalizeArticle(raw string, preserveHyphen bool) string {
	return normalize.NormalizeArticle(raw, normalize.ArticleOptions{PreserveHyphen: preserveHyphen})
}

func (s *Service) NormalizeName(raw string) string {
	return normalize.NormalizeName(raw)
}

type sourceIndex struct {
	byArticle map[string][]domain.Item
	noArticle []domain.Item
}

func indexSource(source []domain.Item, opts Options) sourceIndex {
	byArticle := map[string][]domain.Item{}
	noArticle := make([]domain.Item, 0)
	for _, item := range source {
		article := normalize.NormalizeArticle(item.Article, normalize.ArticleOptions{PreserveHyphen: opts.PreserveHyphen})
		item.Article = article
		if article == "" {
			noArticle = append(noArticle, item)
			continue
		}
		for _, key := range articleKeys(article, opts.PrefixAliases) {
			byArticle[key] = append(byArticle[key], item)
		}
	}
	return sourceIndex{byArticle: byArticle, noArticle: noArticle}
}

func articleKeys(article string, prefixes []string) []string {
	if article == "" {
		return nil
	}
	keys := []string{article}
	for _, prefix := range prefixes {
		switch {
		case strings.HasPrefix(article, prefix) && digitsPattern.MatchString(article[len(prefix):]):
			keys = append(keys, article[len(prefix):])
		case digitsPattern.MatchString(article):
			keys = append(keys, prefix+article)
		}
	}
	return keys
}

func resolve(blank domain.Item, index sourceIndex, opts Options) domain.Result {
	article := normalize.NormalizeArticle(blank.Article, normalize.ArticleOptions{PreserveHyphen: opts.PreserveHyphen})
	candidates := uniqueItems(lookup(index.byArticle, articleKeys(article, opts.PrefixAliases)))
	if len(candidates) == 0 {
		fallback, ok := chooseNameFallback(index.noArticle, blank.Name, blank.Volume)
		if !ok {
			return domain.Result{BlankItemID: blank.ID, Category: domain.CategoryNotInSource, Reasons: domain.Reasons{Source: "none"}}
		}
		return domain.Result{
			BlankItemID:  blank.ID,
			SourceItemID: fallback.item.ID,
			Category:     domain.CategoryNeedsDecision,
			Score:        fallback.score,
			Reasons:      domain.Reasons{Article: "missing", Name: "similar", Source: "name"},
			CandidateIDs: []string{fallback.item.ID},
		}
	}
	ids := make([]string, 0, len(candidates))
	for _, c := range candidates {
		ids = append(ids, c.ID)
	}
	slices.Sort(ids)
	chosen, _ := chooseCandidate(candidates, blank.Name, blank.Volume)
	result := domain.Result{
		BlankItemID:  blank.ID,
		SourceItemID: chosen.item.ID,
		Score:        chosen.score,
		CandidateIDs: ids,
		Reasons: domain.Reasons{
			Article: articleReason(article, chosen.item.Article, opts.PrefixAliases),
			Name:    "similar",
			Source:  "article",
		},
	}
	if len(candidates) > 1 {
		result.Reasons.Duplicates = "chosen_best"
	}
	if opts.Mode == domain.ModeSmart && len(candidates) > 1 {
		second := secondScore(candidates, blank, chosen.item.ID)
		if chosen.score < 0.85 || chosen.score-second < 0.10 {
			result.Category = domain.CategoryNeedsDecision
			result.Reasons.Duplicates = "needs_choice"
			return result
		}
	}
	if volumesConflict(blank, chosen.item) || chosen.score < nameMatchThreshold {
		result.Category = domain.CategoryCheckNameOrVolume
		if volumesConflict(blank, chosen.item) {
			result.Reasons.Volume = "conflict"
		}
		if chosen.score < nameMatchThreshold {
			result.Reasons.Name = "different"
		}
		return result
	}
	result.Category = domain.CategoryToOrder
	return result
}

func lookup(byArticle map[string][]domain.Item, keys []string) []domain.Item {
	out := make([]domain.Item, 0)
	for _, key := range keys {
		out = append(out, byArticle[key]...)
	}
	return out
}

func uniqueItems(items []domain.Item) []domain.Item {
	seen := map[string]bool{}
	out := make([]domain.Item, 0, len(items))
	for _, item := range items {
		id := item.ID
		if id == "" {
			id = item.Article + ":" + item.Name
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, item)
	}
	return out
}

func articleReason(blank, source string, prefixes []string) string {
	if blank != "" && blank == source {
		return "exact"
	}
	for _, key := range articleKeys(blank, prefixes) {
		if key == source {
			return "alias"
		}
	}
	return "exact"
}

func secondScore(candidates []domain.Item, blank domain.Item, winnerID string) float64 {
	best := -1.0
	for _, item := range candidates {
		if item.ID == winnerID {
			continue
		}
		score := volumeAwareSimilarity(blank.Name, item.Name, blank.Volume)
		if score > best {
			best = score
		}
	}
	return best
}
