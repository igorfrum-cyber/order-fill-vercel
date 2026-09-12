export const NORTH_CITIES = [
  { key: "tyumen", label: "Тюмень" },
  { key: "surgut", label: "Сургут" },
  { key: "nizhnevartovsk", label: "Вартовск" },
  { key: "urengoy", label: "Уренгой" },
];

const NORTH_ALLOCATION_ORDER = ["nizhnevartovsk", "urengoy", "surgut"];
const NORTH_TRANSFER_DISPLAY_ORDER = ["surgut", "nizhnevartovsk", "urengoy"];

export function normalizeNorthResult(report, job) {
  return {
    hasTyumenSource: Boolean(report.has_tyumen_source || report.hasTyumenSource),
    uploadedCities: report.uploaded_cities || report.uploadedCities || [],
    planRows: report.plan_rows || report.planRows || [],
    transfers: report.transfers || [],
    confirmationGroups: report.confirmation_groups || report.confirmationGroups || [],
    summary: report.summary || { kind: job.brand || "" },
  };
}

export function formatNorthQuantity(value) {
  const number = Number(value || 0);
  if (!Number.isFinite(number) || number <= 0) return "";
  if (Number.isInteger(number)) return String(number);
  return String(Number(number.toFixed(2)));
}

export function formatNorthCommentQuantity(value) {
  const number = Math.round(Number(value || 0));
  return Number.isFinite(number) && number > 0 ? String(number) : "";
}

export function supplierUnitsFromPieces(quantity, unitSize = 1) {
  const number = Number(quantity || 0);
  const size = Number(unitSize || 1);
  if (!Number.isFinite(number) || number <= 0) return null;
  if (!Number.isFinite(size) || size <= 1) return Number(number.toFixed(2));
  return Math.ceil(number / size);
}

export function demandPiecesFromSupplierUnits(quantity, unitSize = 1) {
  const number = Number(quantity || 0);
  const size = Number(unitSize || 1);
  if (!Number.isFinite(number) || number <= 0) return 0;
  return Number((number * (Number.isFinite(size) && size > 0 ? size : 1)).toFixed(2));
}

export function northTransferParts(row) {
  const quantities = new Map((row.cities || []).map((city) => [city.key, Number(city.quantity || 0)]));
  return NORTH_TRANSFER_DISPLAY_ORDER
    .map((cityKey) => {
      const city = NORTH_CITIES.find((item) => item.key === cityKey);
      return city ? { ...city, quantity: quantities.get(city.key) || 0 } : null;
    })
    .filter(Boolean);
}

export function northSupplierOrderText(row, actual) {
  const actualRounded = Math.round(Number(actual || 0));
  const neededRounded = Math.round(Number(row.supplierNeed || 0));
  const extraRounded = Math.max(0, actualRounded - neededRounded);
  const unitNote = Number(row.supplierUnitSize || 1) > 1 ? ` коробок по ${Number(row.supplierUnitSize)}` : "";
  if (extraRounded > 0) {
    return `${neededRounded} + ${extraRounded} (до минимального) = ${actualRounded}${unitNote}`;
  }
  return `${formatNorthQuantity(actual)}${unitNote}`;
}

export function supplierPartsForNorthActual(row, actualValue = row.actualSupplierOrder) {
  const actual = demandPiecesFromSupplierUnits(actualValue, row.supplierUnitSize);
  const northParts = (row.supplierParts || []).filter((part) => part.key !== "tyumen");
  const northNeed = northParts.reduce((sum, part) => sum + Number(part.quantity || 0), 0);
  const tyumenQuantity = Math.max(0, actual - northNeed);
  return [
    ...(tyumenQuantity > 0 ? [{ key: "tyumen", label: "Тюмень", quantity: Number(tyumenQuantity.toFixed(2)) }] : []),
    ...northParts,
  ];
}

export function northPlanComment(row, actualValue = row.actualSupplierOrder) {
  const lines = [];
  const actual = Number(actualValue || 0);
  const supplierParts = supplierPartsForNorthActual(row, actualValue);
  const tyumenSupplier = supplierParts.find((part) => part.key === "tyumen");
  const transferParts = northTransferParts(row);

  if (actual > 0) lines.push(`Заказать у поставщика: ${northSupplierOrderText(row, actual)}`);
  for (const part of transferParts) {
    if (Number(part.quantity || 0) > 0) lines.push(`Отправить в ${part.label}: ${formatNorthQuantity(part.quantity)}`);
  }
  if (Number(tyumenSupplier?.quantity || 0) > 0) {
    lines.push(`Оставить в Тюмени: ${formatNorthCommentQuantity(tyumenSupplier.quantity)}`);
  }
  if (!lines.length && row.northNeed > 0) lines.push("Закрывается остатком Тюмени");
  return lines.join("\n");
}

export function nearestNorthMultiple(value, multiple) {
  const number = Number(value || 0);
  const step = Math.round(Number(multiple));
  if (!Number.isFinite(number) || number <= 0) return "";
  if (!Number.isFinite(step) || step <= 0) return Number(number.toFixed(2));
  const lower = Math.floor(number / step) * step;
  const upper = Math.ceil(number / step) * step;
  if (lower <= 0) return upper;
  return upper - number <= number - lower ? upper : lower;
}

export function northTyumenFreeStock(stock, inTransit, plannedOrder, target, warehouseStock = null, warehouseTransit = null) {
  const free = Math.max(0, Number(stock) + Number(inTransit) + Number(plannedOrder) - Number(target));
  return warehouseStock == null ? free : Math.min(free, Math.max(0, Number(warehouseStock) + (warehouseTransit || 0) + Number(plannedOrder)));
}

function northBrand(summary = {}) {
  return summary.brand || summary.kind || "";
}

function northBoxStep(brand, row = {}) {
  const box = Number(row.blankBoxSize);
  const boxBrand = !["christina", "klapp", "novacutan", "sothys", "skin_synergy"].includes(brand);
  return boxBrand && Number.isFinite(box) && box > 0 ? Math.ceil(box) : 1;
}

/** North supplier rounding from origin/main defaultNorthActualSupplierOrder, not blank AdjustQuantity. */
export function defaultNorthActual(row, supplierNeed, summary = {}) {
  if (supplierNeed <= 0) return "";
  const brand = northBrand(summary);
  const variant = summary.variant || "";
  if (brand === "christina" || variant === "home" || variant === "proff") {
    return Math.ceil(Number(supplierNeed) / 3 - 1e-10) * 3;
  }
  if (brand === "novacutan") {
    const minimum = Number(row.novacutanMinimum || 100);
    const base = Math.max(Number(supplierNeed), minimum);
    if (Number(row.supplierUnitSize || 1) > 1) return Number(base.toFixed(2));
    return Math.round(base / 10) * 10;
  }
  if (brand === "klapp") return nearestNorthMultiple(supplierNeed, 3);
  const step = northBoxStep(brand, row);
  return Math.ceil(Number(supplierNeed) / step - 1e-10) * step;
}

export function recalculateNorthRow(row, quantities) {
  const cityMap = new Map(Object.entries(quantities).map(([key, value]) => [key, Number(value || 0)]));
  const tyumenUploadedOrder = Number(cityMap.get("tyumen") || 0);
  const tyumenPlannedOrder = cityMap.has("tyumen") ? tyumenUploadedOrder : Number(row.tyumenPlannedOrder || 0);
  const tyumenSupplierNeed = Math.max(0, tyumenPlannedOrder);
  const supplierUnitSize = Number(row.supplierUnitSize || 1);
  const warehouseStock = row.hasWarehouseStock ? Number(row.warehouseStock || 0) : null;
  const warehouseTransit = row.hasWarehouseStock ? Number(row.warehouseTransit || 0) : null;
  const tyumenFree = northTyumenFreeStock(row.tyumenStock || 0, row.tyumenInTransit || 0, tyumenPlannedOrder, row.tyumenTarget || 0, warehouseStock, warehouseTransit);
  let freeLeft = tyumenFree;
  const supplierParts = [];
  const tyumenParts = [];
  let northNeed = 0;
  let supplierNorthNeed = 0;
  let fromTyumen = 0;
  const cities = NORTH_CITIES
    .map((city) => ({ key: city.key, label: city.label, quantity: Number((cityMap.get(city.key) || 0).toFixed(2)) }))
    .filter((city) => city.quantity > 0);

  if (tyumenSupplierNeed > 0) supplierParts.push({ key: "tyumen", label: "Тюмень", quantity: Number(tyumenSupplierNeed.toFixed(2)) });

  for (const cityKey of NORTH_ALLOCATION_ORDER) {
    const city = NORTH_CITIES.find((item) => item.key === cityKey);
    if (!city) continue;
    const quantity = Number(cityMap.get(city.key) || 0);
    if (quantity <= 0) continue;
    northNeed += quantity;
    const fromTyumenPart = Math.min(quantity, freeLeft);
    const fromSupplierPart = quantity - fromTyumenPart;
    freeLeft -= fromTyumenPart;
    fromTyumen += fromTyumenPart;
    supplierNorthNeed += fromSupplierPart;
    if (fromTyumenPart > 0) tyumenParts.push({ key: city.key, label: city.label, quantity: Number(fromTyumenPart.toFixed(2)) });
    if (fromSupplierPart > 0) supplierParts.push({ key: city.key, label: city.label, quantity: Number(fromSupplierPart.toFixed(2)) });
  }

  const supplierDemandNeed = Number((tyumenSupplierNeed + supplierNorthNeed).toFixed(2));
  const supplierNeed = supplierUnitsFromPieces(supplierDemandNeed, supplierUnitSize) || 0;
  return {
    ...row,
    northNeed: Number(northNeed.toFixed(2)),
    cities,
    tyumenFree: Number(tyumenFree.toFixed(2)),
    fromTyumen: Number(fromTyumen.toFixed(2)),
    supplierNorthNeed: Number(supplierNorthNeed.toFixed(2)),
    supplierDemandNeed,
    supplierNeed,
    supplierParts,
    tyumenParts,
  };
}
