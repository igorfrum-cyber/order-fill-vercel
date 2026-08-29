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

export function defaultNorthActual(row, supplierNeed, summary = {}) {
  if (supplierNeed <= 0) return "";
  if (summary.kind === "klapp") return nearestNorthMultiple(supplierNeed, 3);
  if (summary.kind !== "novacutan") return Number(supplierNeed.toFixed(2));
  const minimum = Number(row.novacutanMinimum || 100);
  return Math.round(Math.max(supplierNeed, minimum) / 10) * 10;
}

export function recalculateNorthRow(row, quantities) {
  const cityMap = new Map(Object.entries(quantities).map(([key, value]) => [key, Number(value || 0)]));
  const tyumenUploadedOrder = Number(cityMap.get("tyumen") || 0);
  const tyumenPlannedOrder = cityMap.has("tyumen") ? tyumenUploadedOrder : Number(row.tyumenPlannedOrder || 0);
  const tyumenSupplierNeed = Math.max(0, tyumenPlannedOrder);
  const supplierUnitSize = Number(row.supplierUnitSize || 1);
  let freeLeft = Math.max(0, Number(row.tyumenStock || 0) + Number(row.tyumenInTransit || 0) + tyumenPlannedOrder - Number(row.tyumenTarget || 0));
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
    tyumenFree: Number(Math.max(0, Number(row.tyumenStock || 0) + Number(row.tyumenInTransit || 0) + tyumenPlannedOrder - Number(row.tyumenTarget || 0)).toFixed(2)),
    fromTyumen: Number(fromTyumen.toFixed(2)),
    supplierNorthNeed: Number(supplierNorthNeed.toFixed(2)),
    supplierDemandNeed,
    supplierNeed,
    supplierParts,
    tyumenParts,
  };
}
