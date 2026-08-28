package matching

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"order-fill/services/document-service/internal/domain"
)

type Item struct {
	Article string
	Name    string
	Rounded int
}

type Candidate struct {
	Item  Item
	Score float64
}

var volumePattern = regexp.MustCompile(`(?i)(\d+(?:[,.]\d+)?)\s*(мл|ml|гр|г|g)\b`)

func Similarity(left string, right string) float64 {
	a := domain.NormalizeName(left)
	b := domain.NormalizeName(right)
	if a == "" || b == "" {
		return 0
	}
	rows := make([]int, len([]rune(b))+1)
	leftRunes := []rune(a)
	rightRunes := []rune(b)
	for i := 1; i <= len(leftRunes); i++ {
		previous := 0
		for j := 1; j <= len(rightRunes); j++ {
			tmp := rows[j]
			if leftRunes[i-1] == rightRunes[j-1] {
				rows[j] = previous + 1
			} else if rows[j-1] > rows[j] {
				rows[j] = rows[j-1]
			}
			previous = tmp
		}
	}
	return (2 * float64(rows[len(rightRunes)])) / float64(len(leftRunes)+len(rightRunes))
}

func VolumeAwareSimilarity(blankName string, sourceName string, blankUnit string) float64 {
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

func ChooseCandidate(candidates []Item, blankName string, blankUnit string) (Candidate, bool) {
	if len(candidates) == 0 {
		return Candidate{}, false
	}
	scored := make([]Candidate, 0, len(candidates))
	for _, item := range candidates {
		scored = append(scored, Candidate{
			Item:  item,
			Score: VolumeAwareSimilarity(blankName, item.Name, blankUnit),
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	return scored[0], true
}

func ChooseNameFallback(candidates []Item, blankName string, blankUnit string) (Candidate, bool) {
	scored := make([]Candidate, 0, len(candidates))
	for _, item := range candidates {
		if item.Article != "" {
			continue
		}
		scored = append(scored, Candidate{
			Item:  item,
			Score: VolumeAwareSimilarity(blankName, item.Name, blankUnit),
		})
	}
	if len(scored) == 0 {
		return Candidate{}, false
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if scored[0].Item.Rounded <= 0 && scored[0].Score >= 0.72 {
		return scored[0], true
	}
	if len(scored) > 1 && scored[0].Score-scored[1].Score < 0.08 {
		return Candidate{Score: scored[0].Score}, false
	}
	if scored[0].Score < 0.72 {
		return Candidate{Score: scored[0].Score}, false
	}
	return scored[0], true
}

func extractVolumeKeys(values ...string) map[string]bool {
	keys := map[string]bool{}
	text := domain.NormalizeHeader(strings.Join(values, " "))
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
