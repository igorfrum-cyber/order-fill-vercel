import { roundHalfUp } from "./../calculation/formulas.js";

/** Whole-piece office top-up to 25% of current stock; never a reverse transfer. */
export function warehouseTransferQuantity(officeStock, warehouseStock) {
  const office = Math.max(0, officeStock);
  const warehouse = Math.max(0, warehouseStock);
  return Math.min(Math.floor(warehouse), Math.max(0, roundHalfUp((office + warehouse) * 0.25) - office));
}

/**
 * Projected surplus after protecting Tyumen demand, capped by stock that can
 * physically pass through the warehouse. Office surplus cannot go north.
 * Transit and planned supply are future availability, not goods ready to ship.
 * Null location data preserves the legacy single-location calculation.
 */
export function northTyumenFreeStock(stock, inTransit, plannedOrder, target, warehouseStock = null, warehouseTransit = null) {
  const free = Math.max(0, stock + inTransit + plannedOrder - target);
  return warehouseStock == null ? free : Math.min(free, Math.max(0, warehouseStock + (warehouseTransit || 0) + plannedOrder));
}
