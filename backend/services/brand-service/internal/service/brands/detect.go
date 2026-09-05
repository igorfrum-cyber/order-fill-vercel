package brands

import (
	"strings"
)

func KeyFromNomenclatureGroup(group string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(group, "ё", "е")))
	switch {
	case strings.Contains(value, "ангио") || strings.Contains(value, "angio"):
		return "angiopharm", true
	case strings.Contains(value, "кристин") || strings.Contains(value, "christina"):
		return "christina", true
	case strings.Contains(value, "klapp") || strings.Contains(value, "клапп"):
		return "klapp", true
	case strings.Contains(value, "skin") && strings.Contains(value, "synerg"):
		return "skin_synergy", true
	case strings.Contains(value, "скин") && strings.Contains(value, "синердж"):
		return "skin_synergy", true
	case strings.Contains(value, "levissim") || strings.Contains(value, "левисим"):
		return "levissime", true
	case strings.Contains(value, "sothys") || strings.Contains(value, "сотис"):
		return "sothys", true
	case strings.Contains(value, "novacutan") || strings.Contains(value, "новакутан"):
		return "novacutan", true
	default:
		return "", false
	}
}

func Detect(group, fileName string) (brand string, variant string, ok bool) {
	brand, ok = KeyFromNomenclatureGroup(group)
	if !ok {
		return "", "", false
	}
	if brand == "christina" {
		variant = christinaVariant(fileName)
	}
	return brand, variant, true
}

func christinaVariant(name string) string {
	value := strings.ToLower(strings.ReplaceAll(name, "ё", "е"))
	proff := strings.Contains(value, "proff") || strings.Contains(value, "prof") || strings.Contains(value, "проф")
	home := strings.Contains(value, "home") || strings.Contains(value, "дом")
	if proff && !home {
		return "PROFF"
	}
	if home && !proff {
		return "HOME"
	}
	return ""
}
