package matching

import (
	"order-fill/backend/services/matching-service/internal/domain"
	"order-fill/backend/services/matching-service/internal/normalize"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

var leviVolume = regexp.MustCompile(`(?i)\d+(?:[.,]\d+)?[\s\p{Z}\x{000B}\x{FEFF}]*(?:мл|мг|г|л)`)
var leviEnglish = regexp.MustCompile(`^[A-Za-z\s\p{Z}\x{000B}\x{FEFF}+().-]+$`)
var leviEnglishSuffix = regexp.MustCompile(`[a-z][a-z0-9]*`)
var leviRussian = regexp.MustCompile(`[а-яё]`)
var leviDescription = regexp.MustCompile(`[^A-Za-z0-9\s\p{Z}\x{000B}\x{FEFF}+().,-]`)

func leviName(v string) string {
	text := comparableChzName(v)
	text = strings.TrimPrefix(strings.TrimPrefix(text, "levissime "), "lev ")
	return strings.ReplaceAll(text, "отбеливающая", "осветляющая")
}
func leviRussianIdentity(v string) string {
	parts := strings.Split(regexp.MustCompile(`(?i)^чз[\s\p{Z}\x{000B}\x{FEFF}]*\+?[\s\p{Z}\x{000B}\x{FEFF}]*`).ReplaceAllString(v, ""), "/")
	if len(parts) > 2 || !leviRussian.MatchString(strings.ToLower(parts[0])) || len(leviVolumeIndices(parts[0])) == 0 {
		return ""
	}
	if len(parts) == 2 && leviDescription.MatchString(parts[1]) {
		return ""
	}
	return strings.Join(strings.Fields(leviEnglishSuffix.ReplaceAllString(leviName(parts[0]), " ")), " ")
}
func leviVolumes(v string) string {
	keys := []string{}
	for _, index := range leviVolumeIndices(v) {
		volume := strings.ToLower(v)[index[0]:index[1]]
		v := volume
		v = strings.ReplaceAll(strings.TrimRight(strings.TrimSpace(v), " /+().,-;:"), ",", ".")
		keys = append(keys, strings.Join(strings.Fields(strings.ReplaceAll(v, "\ufeff", "")), ""))
	}
	slices.Sort(keys)
	return strings.Join(slices.Compact(keys), ";")
}
func leviEnglishIdentity(v string) string {
	parts := strings.Split(v, "/")
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if p == "" || p[0] < 'A' || p[0] > 'z' || p[0] > 'Z' && p[0] < 'a' {
			continue
		}
		if loc := leviVolumeIndices(p); len(loc) > 0 {
			p = strings.TrimSpace(p[:loc[0][0]])
		}
		if !leviEnglish.MatchString(p) {
			return ""
		}
		text := normalize.NormalizeHeader(p)
		if len(strings.Fields(text)) < 3 {
			return ""
		}
		return text
	}
	return ""
}
func leviEnglishMatch(a, b string) bool {
	id, v := leviEnglishIdentity(a), leviVolumes(a)
	return id != "" && v != "" && id == leviEnglishIdentity(b) && v == leviVolumes(b)
}
func mergeLevissime(normals, clones []domain.Item) []ChzMerge {
	out := []ChzMerge{}
	used := map[string]bool{}
	for _, target := range normals {
		if used[target.ID] {
			continue
		}
		id := leviRussianIdentity(target.Name)
		ids := []string{}
		for _, normal := range normals {
			if normal.ID != target.ID && !used[normal.ID] && id != "" && id == leviRussianIdentity(normal.Name) {
				ids = append(ids, normal.ID)
				used[normal.ID] = true
			}
		}
		for _, clone := range clones {
			if !used[clone.ID] && leviVolumes(target.Name) == leviVolumes(clone.Name) && (id != "" && id == leviRussianIdentity(clone.Name) || leviEnglishMatch(target.Name, clone.Name) || Similarity(leviName(target.Name), leviName(clone.Name)) >= 0.9) {
				ids = append(ids, clone.ID)
				used[clone.ID] = true
			}
		}
		if len(ids) > 0 {
			out = append(out, ChzMerge{TargetID: target.ID, CloneIDs: ids})
		}
	}
	return out
}

func leviVolumeIndices(value string) [][]int {
	text := strings.ToLower(value)
	out := [][]int{}
	for _, loc := range leviVolume.FindAllStringIndex(text, -1) {
		if loc[1] < len(text) {
			r, _ := utf8.DecodeRuneInString(text[loc[1]:])
			if r >= 'а' && r <= 'я' || r == 'ё' {
				continue
			}
		}
		out = append(out, loc)
	}
	return out
}
