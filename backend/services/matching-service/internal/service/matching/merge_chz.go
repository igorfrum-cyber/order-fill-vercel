package matching

import (
	"regexp"
	"strings"

	"order-fill/backend/services/matching-service/internal/domain"
	"order-fill/backend/services/matching-service/internal/normalize"
)

const chzNameThreshold = 0.9

var (
	chzPrefixPattern = regexp.MustCompile(`^чз\s+`)
	chzWordPattern   = regexp.MustCompile(`\bчз\b`)
	chzPlusPattern   = regexp.MustCompile(`^чз\s*\+`)
)

type ChzMerge struct {
	TargetID      string
	CloneIDs      []string
	NeedsDecision bool
}

func (s *Service) MergeChz(items []domain.Item, opts Options) []ChzMerge {
	grouped := map[string][]domain.Item{}
	order := make([]string, 0)
	for _, item := range items {
		article := normalize.NormalizeArticle(item.Article, normalize.ArticleOptions{PreserveHyphen: opts.PreserveHyphen})
		if article == "" {
			continue
		}
		item.Article = article
		if _, seen := grouped[article]; !seen {
			order = append(order, article)
		}
		grouped[article] = append(grouped[article], item)
	}
	out := make([]ChzMerge, 0)
	for _, article := range order {
		out = append(out, mergeArticleChz(grouped[article], opts)...)
	}
	return out
}

func mergeArticleChz(group []domain.Item, opts Options) []ChzMerge {
	normals := make([]domain.Item, 0, len(group))
	clones := make([]domain.Item, 0, len(group))
	for _, item := range group {
		if item.ChestnyZnak || isChzName(item.Name) {
			clones = append(clones, item)
			continue
		}
		normals = append(normals, item)
	}
	if len(normals) == 0 || len(clones) == 0 {
		return nil
	}
	if opts.Mode == domain.ModeSmart && len(normals) > 1 {
		return mergeChzSmart(normals, clones)
	}
	return mergeChzStandard(normals[0], clones)
}

func mergeChzStandard(target domain.Item, clones []domain.Item) []ChzMerge {
	ids := make([]string, 0, len(clones))
	for _, clone := range clones {
		if chzAttachOK(target, clone) {
			ids = append(ids, clone.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return []ChzMerge{{TargetID: target.ID, CloneIDs: ids}}
}

func mergeChzSmart(normals, clones []domain.Item) []ChzMerge {
	out := make([]ChzMerge, 0, len(clones))
	byTarget := map[string][]string{}
	targetOrder := make([]string, 0)
	for _, clone := range clones {
		best, second := scoredNormals(normals, clone)
		if best.score < 0.85 || (second >= 0 && best.score-second < 0.10) {
			out = append(out, ChzMerge{CloneIDs: []string{clone.ID}, NeedsDecision: true})
			continue
		}
		if !chzAttachOK(best.item, clone) {
			out = append(out, ChzMerge{CloneIDs: []string{clone.ID}, NeedsDecision: true})
			continue
		}
		if _, seen := byTarget[best.item.ID]; !seen {
			targetOrder = append(targetOrder, best.item.ID)
		}
		byTarget[best.item.ID] = append(byTarget[best.item.ID], clone.ID)
	}
	for _, id := range targetOrder {
		out = append(out, ChzMerge{TargetID: id, CloneIDs: byTarget[id]})
	}
	return out
}

func scoredNormals(normals []domain.Item, clone domain.Item) (scored, float64) {
	best := scored{item: normals[0], score: chzScore(normals[0], clone)}
	second := -1.0
	for _, item := range normals[1:] {
		score := chzScore(item, clone)
		if score > best.score {
			second = best.score
			best = scored{item: item, score: score}
			continue
		}
		if score > second {
			second = score
		}
	}
	return best, second
}

func chzScore(normal, clone domain.Item) float64 {
	return volumeAwareSimilarity(comparableChzName(normal.Name), comparableChzName(clone.Name), normal.Volume)
}

func chzAttachOK(target, clone domain.Item) bool {
	if Similarity(comparableChzName(target.Name), comparableChzName(clone.Name)) < chzNameThreshold {
		return false
	}
	return !volumesConflict(target, clone) && !formsConflict(target, clone)
}

func isChzName(value string) bool {
	header := normalize.NormalizeHeader(value)
	return strings.HasPrefix(header, "чз ") && !chzPlusPattern.MatchString(header)
}

func comparableChzName(value string) string {
	text := chzPrefixPattern.ReplaceAllString(normalize.NormalizeHeader(value), "")
	text = chzWordPattern.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}
