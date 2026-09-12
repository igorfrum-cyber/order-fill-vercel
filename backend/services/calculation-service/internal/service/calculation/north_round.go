package calculation

import (
	"math"

	"order-fill/backend/services/calculation-service/internal/domain"
)

// ChristinaNorthSupplierQuantity matches origin/main christinaNorthSupplierQuantity.
// ok is false when need <= 0 (JS null).
func ChristinaNorthSupplierQuantity(need float64) (float64, bool) {
	if need <= 0 {
		return 0, false
	}
	return math.Ceil(need/3-1e-10) * 3, true
}

// DefaultNorthActualSupplierOrder matches origin/main defaultNorthActualSupplierOrder.
// need <= 0 returns 0 (JS null). This is north rounding, not blank AdjustQuantity.
func DefaultNorthActualSupplierOrder(brand, variant string, pos domain.NorthSupplierPosition, need float64) float64 {
	if need <= 0 {
		return 0
	}
	if brand == "christina" || variant == "home" || variant == "proff" {
		qty, _ := ChristinaNorthSupplierQuantity(need)
		return qty
	}
	if brand == "novacutan" {
		return novacutanSupplierOrderQuantity(pos, need)
	}
	if brand == "klapp" {
		v, ok := nearestMultipleValue(need, 3)
		if ok {
			return v
		}
		return 0
	}
	step := 1.0
	rule := brandAdjustment(brand)
	if rule.kind == AdjustmentBox && pos.HasBoxSize && !math.IsNaN(pos.BlankBoxSize) && !math.IsInf(pos.BlankBoxSize, 0) && pos.BlankBoxSize > 0 {
		step = math.Ceil(pos.BlankBoxSize)
	}
	return math.Ceil(need/step-1e-10) * step
}

func novacutanSupplierOrderQuantity(pos domain.NorthSupplierPosition, need float64) float64 {
	if need <= 0 {
		return 0
	}
	minimum := pos.NovacutanMinimum
	if minimum == 0 || math.IsNaN(minimum) {
		minimum = 100
	}
	base := max(need, minimum)
	unit := pos.SupplierUnitSize
	if unit == 0 || math.IsNaN(unit) {
		unit = 1
	}
	if unit > 1 {
		return math.Round(base*100) / 100
	}
	return roundSupplierPack10(base)
}
