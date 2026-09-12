package calculation

import "math"

// RoundHalfUp matches origin/main workbookProcessor.roundHalfUp.
func RoundHalfUp(value float64) float64 {
	return math.Floor(value + 0.5)
}

// NorthTyumenFreeStock matches origin/main northTyumenFreeStock.
// hasWarehouseStock false is JS warehouseStock == null (legacy single-location).
func NorthTyumenFreeStock(stock, inTransit, plannedOrder, target, warehouseStock, warehouseTransit float64, hasWarehouseStock bool) float64 {
	free := max(0, stock+inTransit+plannedOrder-target)
	if !hasWarehouseStock {
		return free
	}
	return min(free, max(0, warehouseStock+warehouseTransit+plannedOrder))
}

// WarehouseTransferQuantity matches origin/main warehouseTransferQuantity.
// Whole-piece office top-up to 25% of current stock; never a reverse transfer.
func WarehouseTransferQuantity(officeStock, warehouseStock float64) float64 {
	office := max(0, officeStock)
	warehouse := max(0, warehouseStock)
	return min(math.Floor(warehouse), max(0, RoundHalfUp((office+warehouse)*0.25)-office))
}

// WarehouseOfficeTarget is roundHalfUp((office+warehouse)*0.25).
func WarehouseOfficeTarget(officeStock, warehouseStock float64) float64 {
	return RoundHalfUp((max(0, officeStock) + max(0, warehouseStock)) * 0.25)
}
