package calculation

import (
	"math"
	"strconv"
)

type Adjustment string

const (
	AdjustmentNone            Adjustment = "none"
	AdjustmentBox             Adjustment = "box"
	AdjustmentMultiple        Adjustment = "multiple"
	AdjustmentNearestMultiple Adjustment = "nearestMultiple"
	AdjustmentMinimum         Adjustment = "minimum"
)

type AdjustedQuantity struct {
	Rounded     int
	Inserted    *float64
	AutoComment string
	BoxAdjusted bool
}

func AdjustQuantity(recommended float64, brand string, orderedFact float64, hasFact bool, boxSizeValue string) AdjustedQuantity {
	qty := recommended
	if hasFact {
		qty = orderedFact
	}
	rule := brandAdjustment(brand)
	if boxSizeValue == "" && rule.multiple > 0 {
		boxSizeValue = strconv.Itoa(rule.multiple)
	}
	result := calculateAdjustedQuantity(qty, rule, boxSizeValue)
	if hasFact {
		result.AutoComment = ""
	}
	return result
}

type brandRule struct {
	kind                    Adjustment
	multiple                int
	comment                 string
	allowSmallPositiveOrder bool
}

func brandAdjustment(brand string) brandRule {
	switch brand {
	case "christina":
		return brandRule{kind: AdjustmentMultiple, multiple: 3, comment: "до кратности 3"}
	case "klapp":
		return brandRule{kind: AdjustmentNearestMultiple, multiple: 3, comment: "до кратности 3"}
	case "novacutan":
		return brandRule{kind: AdjustmentNone}
	case "sothys", "skin_synergy":
		return brandRule{kind: AdjustmentNone}
	default:
		return brandRule{kind: AdjustmentBox, comment: "до коробки"}
	}
}

func calculateAdjustedQuantity(recommended float64, rule brandRule, boxSizeValue string) AdjustedQuantity {
	rounded := int(math.Floor(recommended + 0.5))
	if !rule.allowSmallPositiveOrder && recommended < 1.5 {
		return AdjustedQuantity{Rounded: rounded}
	}
	if rounded <= 0 {
		return AdjustedQuantity{Rounded: rounded}
	}
	switch rule.kind {
	case AdjustmentNone:
		v := float64(rounded)
		return AdjustedQuantity{Rounded: rounded, Inserted: &v}
	case AdjustmentMultiple:
		return calculateMultipleAdjustedQuantity(rounded, rule.multiple, rule.comment)
	case AdjustmentNearestMultiple:
		return calculateNearestMultipleAdjustedQuantity(rounded, rule.multiple, rule.comment)
	case AdjustmentMinimum:
		minimum, ok := parseNumber(boxSizeValue)
		minInt := int(math.Round(minimum))
		if ok && minInt > 0 && rounded < minInt {
			v := float64(minInt)
			return AdjustedQuantity{Rounded: rounded, Inserted: &v, AutoComment: rule.comment, BoxAdjusted: true}
		}
		v := float64(rounded)
		return AdjustedQuantity{Rounded: rounded, Inserted: &v}
	default:
		boxSize, ok := parseNumber(boxSizeValue)
		if !ok || boxSize <= 0 {
			v := float64(rounded)
			return AdjustedQuantity{Rounded: rounded, Inserted: &v}
		}
		return calculateMultipleAdjustedQuantity(rounded, int(math.Round(boxSize)), rule.comment)
	}
}

func calculateMultipleAdjustedQuantity(rounded, multiple int, comment string) AdjustedQuantity {
	step := int(math.Round(float64(multiple)))
	if step <= 0 || rounded%step == 0 {
		v := float64(rounded)
		return AdjustedQuantity{Rounded: rounded, Inserted: &v}
	}
	lower := int(math.Floor(float64(rounded)/float64(step))) * step
	upper := int(math.Ceil(float64(rounded)/float64(step))) * step
	upPercent := float64(upper-rounded) / float64(rounded)
	downPercent := math.Inf(1)
	if lower > 0 {
		downPercent = float64(rounded-lower) / float64(rounded)
	}
	if upPercent > 0 && upPercent <= 0.15 {
		v := float64(upper)
		return AdjustedQuantity{Rounded: rounded, Inserted: &v, AutoComment: comment, BoxAdjusted: true}
	}
	if downPercent > 0 && downPercent <= 0.05 {
		v := float64(lower)
		return AdjustedQuantity{Rounded: rounded, Inserted: &v, AutoComment: comment, BoxAdjusted: true}
	}
	v := float64(rounded)
	return AdjustedQuantity{Rounded: rounded, Inserted: &v}
}

func calculateNearestMultipleAdjustedQuantity(rounded, multiple int, comment string) AdjustedQuantity {
	inserted, ok := nearestMultipleValue(float64(rounded), multiple)
	if !ok {
		return AdjustedQuantity{Rounded: rounded}
	}
	adjusted := inserted != float64(rounded)
	result := AdjustedQuantity{Rounded: rounded, Inserted: &inserted, BoxAdjusted: adjusted}
	if adjusted {
		result.AutoComment = comment
	}
	return result
}

func nearestMultipleValue(value float64, multiple int) (float64, bool) {
	step := int(math.Round(float64(multiple)))
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
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

func parseNumber(text string) (float64, bool) {
	if text == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
