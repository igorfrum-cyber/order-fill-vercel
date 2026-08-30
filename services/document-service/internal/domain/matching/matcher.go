package matching

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"order-fill/services/document-service/internal/domain/normalize"
)

type Item struct {
	// Ref is an opaque caller-owned identifier so the winner can be traced back
	// to the row it came from.
	Ref     int
	Article string
	Name    string
	Rounded int
}

type Candidate struct {
	Item  Item
	Score float64
}

var volumePattern = regexp.MustCompile(`(?i)(\d+(?:[,.]\d+)?)\s*(мл|ml|гр|г|g)\b`)

var lcsPool = sync.Pool{New: func() any {
	buf := make([]int, 0, 64)
	return &buf
}}

func Similarity(left string, right string) float64 {
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
	scored := make([]Candidate, len(candidates))
	score := func(i int) {
		scored[i] = Candidate{
			Item:  candidates[i],
			Score: VolumeAwareSimilarity(blankName, candidates[i].Name, blankUnit),
		}
	}
	if len(candidates) < 16 {
		for i := range candidates {
			score(i)
		}
	} else {
		runWorkers(len(candidates), score)
	}
	best := 0
	for i := 1; i < len(scored); i++ {
		if scored[i].Score > scored[best].Score {
			best = i
		}
	}
	return scored[best], true
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
