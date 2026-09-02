package brand

import (
	"math"
	"strconv"
	"strings"

	"order-fill/services/document-service/internal/domain/normalize"
)

type Adjustment string

const (
	AdjustmentNone            Adjustment = "none"
	AdjustmentBox             Adjustment = "box"
	AdjustmentMultiple        Adjustment = "multiple"
	AdjustmentNearestMultiple Adjustment = "nearestMultiple"
	AdjustmentMinimum         Adjustment = "minimum"
)

type RuleConfig struct {
	Key                     string
	Label                   string
	Adjustment              Adjustment
	Multiple                int
	AdjustmentLabel         string
	AdjustmentComment       string
	PreserveArticleHyphen   bool
	ArticlePrefixAliases    []string
	BlankQuantityHeader     string
	BlankBoxHeader          string
	RequireUnit             *bool
	AllowSmallPositiveOrder bool
	// BlankLayout names a non-generic blank structure. An empty value means the
	// standard article/name/quantity table.
	BlankLayout string
}

type AdjustedQuantity struct {
	Rounded     int
	Inserted    *float64
	AutoComment string
	BoxAdjusted bool
}

var rules = map[string]RuleConfig{
	"angiopharm": {
		Key:               "angiopharm",
		Label:             "ANGIOPHARM",
		Adjustment:        AdjustmentBox,
		AdjustmentLabel:   "Шт. в коробке",
		AdjustmentComment: "до коробки",
	},
	"christina": {
		Key:               "christina",
		Label:             "CHRISTINA",
		Adjustment:        AdjustmentMultiple,
		Multiple:          3,
		AdjustmentLabel:   "Кратность",
		AdjustmentComment: "до кратности 3",
	},
	"levissime": {
		Key:                  "levissime",
		Label:                "LeviSsime",
		Adjustment:           AdjustmentBox,
		AdjustmentLabel:      "Кол-во в уп.",
		AdjustmentComment:    "до коробки",
		ArticlePrefixAliases: []string{"MT"},
		BlankQuantityHeader:  "order",
		BlankBoxHeader:       "packageQuantity",
	},
	"sothys": {
		Key:                   "sothys",
		Label:                 "SOTHYS",
		Adjustment:            AdjustmentNone,
		AdjustmentLabel:       "Без округления",
		PreserveArticleHyphen: true,
		BlankLayout:           "splitVariants",
	},
	"novacutan": {
		Key:             "novacutan",
		Label:           "NOVACUTAN",
		Adjustment:      AdjustmentNone,
		AdjustmentLabel: "Мин. заказ",
		BlankLayout:     "novacutan",
	},
	"skin_synergy": {
		Key:                 "skin_synergy",
		Label:               "Skin Synergy",
		Adjustment:          AdjustmentNone,
		AdjustmentLabel:     "Без округления",
		BlankQuantityHeader: "exactQuantity",
		RequireUnit:         boolPtr(false),
	},
	"klapp": {
		Key:                 "klapp",
		Label:               "KLAPP",
		Adjustment:          AdjustmentNearestMultiple,
		Multiple:            3,
		AdjustmentLabel:     "Кратность",
		AdjustmentComment:   "до кратности 3",
		BlankQuantityHeader: "order",
	},
}

func Rule(brand string) RuleConfig {
	if rule, ok := rules[brand]; ok {
		return rule
	}
	return rules["angiopharm"]
}

// KeyFromNomenclatureGroup maps the 1C filter caption "Номенклатура В группе …"
// onto the brand key used by Rule.
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

func ArticleNormalizeOptions(rule RuleConfig) normalize.ArticleOptions {
	return normalize.ArticleOptions{PreserveHyphen: rule.PreserveArticleHyphen}
}

func CategoryCoefficient(value string, rule RuleConfig) float64 {
	category := normalize.NormalizeCategory(value)
	if rule.Label == "NOVACUTAN" {
		if category == "C" {
			return 1.5
		}
		if category == "A+" || category == "A" || category == "B" {
			return 2
		}
	}
	switch category {
	case "A+":
		return 2
	case "A":
		return 1.75
	case "B":
		return 1.5
	case "C":
		return 1
	default:
		return 1
	}
}

func CalculateAdjustedQuantity(recommended float64, rule RuleConfig, boxSizeValue string) AdjustedQuantity {
	rounded := normalize.RoundHalfUp(recommended)
	if !rule.AllowSmallPositiveOrder && recommended < 1.5 {
		return AdjustedQuantity{Rounded: rounded}
	}
	if rounded <= 0 {
		return AdjustedQuantity{Rounded: rounded}
	}

	switch rule.Adjustment {
	case AdjustmentNone:
		return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(rounded))}
	case AdjustmentMultiple:
		return calculateMultipleAdjustedQuantity(rounded, rule.Multiple, rule.AdjustmentComment)
	case AdjustmentNearestMultiple:
		return calculateNearestMultipleAdjustedQuantity(rounded, rule.Multiple, rule.AdjustmentComment)
	case AdjustmentMinimum:
		minimumValue, ok := normalize.ParseNumber(boxSizeValue)
		minimum := int(math.Round(minimumValue))
		if ok && minimum > 0 && rounded < minimum {
			return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(minimum)), AutoComment: rule.AdjustmentComment, BoxAdjusted: true}
		}
		return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(rounded))}
	default:
		boxSize, ok := normalize.ParseNumber(boxSizeValue)
		if !ok || boxSize <= 0 {
			return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(rounded))}
		}
		return calculateMultipleAdjustedQuantity(rounded, int(math.Round(boxSize)), rule.AdjustmentComment)
	}
}

func AdjustQuantityForBrand(recommended float64, brand string, boxSizeValue string) AdjustedQuantity {
	rule := Rule(brand)
	if boxSizeValue == "" && rule.Multiple > 0 {
		boxSizeValue = strconv.Itoa(rule.Multiple)
	}
	return CalculateAdjustedQuantity(recommended, rule, boxSizeValue)
}

func calculateMultipleAdjustedQuantity(rounded int, multiple int, comment string) AdjustedQuantity {
	step := int(math.Round(float64(multiple)))
	if step <= 0 || rounded%step == 0 {
		return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(rounded))}
	}
	lower := int(math.Floor(float64(rounded)/float64(step))) * step
	upper := int(math.Ceil(float64(rounded)/float64(step))) * step
	upPercent := float64(upper-rounded) / float64(rounded)
	downPercent := math.Inf(1)
	if lower > 0 {
		downPercent = float64(rounded-lower) / float64(rounded)
	}
	if upPercent > 0 && upPercent <= 0.15 {
		return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(upper)), AutoComment: comment, BoxAdjusted: true}
	}
	if downPercent > 0 && downPercent <= 0.05 {
		return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(lower)), AutoComment: comment, BoxAdjusted: true}
	}
	return AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(float64(rounded))}
}

func calculateNearestMultipleAdjustedQuantity(rounded int, multiple int, comment string) AdjustedQuantity {
	inserted, ok := nearestMultipleValue(float64(rounded), multiple)
	if !ok {
		return AdjustedQuantity{Rounded: rounded}
	}
	adjusted := inserted != float64(rounded)
	result := AdjustedQuantity{Rounded: rounded, Inserted: floatPtr(inserted), BoxAdjusted: adjusted}
	if adjusted {
		result.AutoComment = comment
	}
	return result
}

func nearestMultipleValue(value float64, multiple int) (float64, bool) {
	step := int(math.Round(float64(multiple)))
	if !isFinitePositive(value) {
		return 0, false
	}
	if step <= 0 {
		return math.Round(value*100) / 100, true
	}
	lower := math.Floor(value/float64(step)) * float64(step)
	upper := math.Ceil(value/float64(step)) * float64(step)
	if lower <= 0 {
		return upper, true
	}
	if upper-value <= value-lower {
		return upper, true
	}
	return lower, true
}

func isFinitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func floatPtr(value float64) *float64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
