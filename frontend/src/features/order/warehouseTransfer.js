/** Whole-piece office top-up to 25% of current stock; never a reverse transfer. Matches origin/main. */
export function roundHalfUp(value) {
  return Math.floor(value + 0.5);
}

export function warehouseTransferQuantity(officeStock, warehouseStock) {
  const office = Math.max(0, officeStock);
  const warehouse = Math.max(0, warehouseStock);
  return Math.min(Math.floor(warehouse), Math.max(0, roundHalfUp((office + warehouse) * 0.25) - office));
}

export function warehouseOfficeTarget(officeStock, warehouseStock) {
  return roundHalfUp((Math.max(0, officeStock) + Math.max(0, warehouseStock)) * 0.25);
}
