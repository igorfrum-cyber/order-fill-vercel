package matching

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"order-fill/backend/services/matching-service/internal/domain"
	"order-fill/backend/services/matching-service/internal/normalize"
)

type scored struct {
	item  domain.Item
	score float64
}

var volumePattern = regexp.MustCompile(`(?i)(\d+(?:[,.]\d+)?)\s*(мл|ml|гр|г|g)\b`)

var lcsPool = sync.Pool{New: func() any {
	buf := make([]int, 0, 64)
	return &buf
}}

func Similarity(left, right string) float64 {
	a := normalize.NormalizeName(left)
	b := normalize.NormalizeName(right)
	if a == "" || b == "" {
		return 0
	}
	leftRunes := []rune(a)
	rightRunes := []rune(b)
	needed := len(rightRunes) + 1
	ptr := lcsPool.Get().(*[]int)
	buf := *ptr
	if cap(buf) < needed {
		buf = make([]int, needed)
	} else {
		buf = buf[:needed]
		clear(buf)
	}
	defer func() {
		*ptr = buf[:0]
		lcsPool.Put(ptr)
	}()
	for i := 1; i <= len(leftRunes); i++ {
		previous := 0
		for j := 1; j <= len(rightRunes); j++ {
			tmp := buf[j]
			if leftRunes[i-1] == rightRunes[j-1] {
				buf[j] = previous + 1
			} else if buf[j-1] > buf[j] {
				buf[j] = buf[j-1]
			}
			previous = tmp
		}
	}
	return (2 * float64(buf[len(rightRunes)])) / float64(len(leftRunes)+len(rightRunes))
}

func volumeAwareSimilarity(blankName, sourceName, blankUnit string) float64 {
	base := Similarity(blankName, sourceName)
	blankVolumes := extractVolumeKeys(blankName, blankUnit)
	sourceVolumes := extractVolumeKeys(sourceName)
	if len(blankVolumes) == 0 || len(sourceVolumes) == 0 {
		return base
	}
	for key := range blankVolumes {
		if sourceVolumes[key] {
			return math.Min(1, base+0.06)
		}
	}
	return math.Max(0, base-0.35)
}

func chooseCandidate(candidates []domain.Item, blankName, blankUnit string) (scored, bool) {
	if len(candidates) == 0 {
		return scored{}, false
	}
	best := scored{item: candidates[0], score: volumeAwareSimilarity(blankName, candidates[0].Name, blankUnit)}
	for _, item := range candidates[1:] {
		score := volumeAwareSimilarity(blankName, item.Name, blankUnit)
		if score > best.score {
			best = scored{item: item, score: score}
		}
	}
	return best, true
}

func chooseNameFallback(candidates []domain.Item, blankName, blankUnit string) (scored, bool) {
	ranked := []scored{}
	for _, item := range candidates {
		if item.Article == "" {
			ranked = append(ranked, scored{item: item, score: volumeAwareSimilarity(blankName, item.Name, blankUnit)})
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
	for _, candidate := range ranked {
		if candidate.item.Rounded <= 0 && candidate.score >= .72 {
			return candidate, true
		}
	}
	if ranked[0].score < .72 || len(ranked) > 1 && ranked[0].score-ranked[1].score < .08 {
		return scored{score: ranked[0].score}, false
	}
	return ranked[0], true
}

func extractVolumeKeys(values ...string) map[string]bool {
	keys := map[string]bool{}
	text := normalize.NormalizeHeader(strings.Join(values, " "))
	for _, match := range volumePattern.FindAllStringSubmatch(text, -1) {
		amount, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64)
		if err != nil {
			continue
		}
		unit := strings.ToLower(match[2])
		normalizedUnit := "гр"
		if unit == "ml" || unit == "мл" {
			normalizedUnit = "мл"
		}
		keys[strconv.FormatFloat(amount, 'f', -1, 64)+":"+normalizedUnit] = true
	}
	return keys
}

func volumesConflict(blank, source domain.Item) bool {
	left := extractConflictVolumeKeys(blank.Name, blank.Volume)
	right := extractConflictVolumeKeys(source.Name, source.Volume)
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	for key := range left {
		if right[key] {
			return false
		}
	}
	return true
}

var formTokens = []string{
	"сыворотка", "serum", "молочко", "лосьон", "lotion", "шампунь", "эмульсия",
	"бальзам", "скраб", "флюид", "тоник", "toner", "маска", "mask", "пенка",
	"спрей", "spray", "масло", "oil", "крем", "cream", "гель", "gel",
}

func extractForm(item domain.Item) string {
	if form := strings.ToLower(strings.TrimSpace(item.Form)); form != "" {
		return form
	}
	text := normalize.NormalizeHeader(item.Name)
	for _, token := range formTokens {
		if strings.Contains(text, token) {
			return token
		}
	}
	return ""
}

func formsConflict(blank, source domain.Item) bool {
	left, right := extractForm(blank), extractForm(source)
	return left != "" && right != "" && left != right
}

var conflictVolumePattern = regexp.MustCompile(`(?i)(\d+(?:[,.]\d+)?)\s*(мл|ml|гр|г|g)(?:\s|$)`)

func extractConflictVolumeKeys(values ...string) map[string]bool {
	keys := map[string]bool{}
	text := normalize.NormalizeHeader(strings.Join(values, " "))
	for _, m := range conflictVolumePattern.FindAllStringSubmatch(text, -1) {
		amount, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
		if err != nil {
			continue
		}
		unit := "гр"
		if m[2] == "мл" || m[2] == "ml" {
			unit = "мл"
		}
		keys[strconv.FormatFloat(amount, 'f', -1, 64)+":"+unit] = true
	}
	return keys
}
