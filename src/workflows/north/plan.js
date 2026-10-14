import { northTyumenFreeStock } from "./../../domain/warehouse/stock.js";
import { supplierUnitsFromPieces } from "./../../domain/order/quantities.js";
import { allocateNorthNeedByCity, defaultNorthActualSupplierOrder, northTotalCityParts } from "./../../domain/north/allocation.js";
import { roundHalfUp } from "./../../domain/calculation/formulas.js";
import { northAvailabilityForTotal, readNorthTyumenAvailability } from "./availability.js";
import { parseNumber } from "./../../shared/text.js";
import { NORTH_CITIES } from "./../../domain/north/cities.js";

export function addNorthTotal(totals, item, city) {
  if (!item.key || item.quantity <= 0) return;
  if (!totals.has(item.key)) {
    totals.set(item.key, {
      key: item.key,
      articleRaw: item.articleRaw,
      name: item.name,
      unit: item.unit,
      baseKey: item.baseKey || item.key,
      variant: item.variant || "",
      variantLabel: item.variantLabel || "",
      totalQuantity: 0,
      cities: new Map(),
    });
  }
  const current = totals.get(item.key);
  current.totalQuantity += item.quantity;
  if (!current.name && item.name) current.name = item.name;
  if (!current.unit && item.unit) current.unit = item.unit;
  if (!current.articleRaw && item.articleRaw) current.articleRaw = item.articleRaw;
  current.cities.set(city.key, (current.cities.get(city.key) || 0) + item.quantity);
}

export function scopedNorthKey(key, variant = "") {
  return variant ? `${variant}:${key}` : key;
}

export function scopedNorthPositions(positions, variant = "", variantLabel = "") {
  if (!variant) return positions;
  return positions.map((position) => ({
    ...position,
    baseKey: position.key,
    key: scopedNorthKey(position.key, variant),
    variant,
    variantLabel,
  }));
}

export function northPlanRowFromTotal(summary, total, position, tyumen = null, tyumenFallbackOrder = 0) {
  const tyumenUploadedOrder = Number(total.cities.get("tyumen") || 0);
  const tyumenPlannedOrder = Number(total.cities.has("tyumen") ? tyumenUploadedOrder : tyumenFallbackOrder);
  const tyumenSupplierNeed = Math.max(0, tyumenPlannedOrder);
  const tyumenStock = Number(tyumen?.stock || 0);
  const tyumenInTransit = Number(tyumen?.inTransit || 0);
  const tyumenTarget = Number(tyumen?.targetStock || 0);
  const tyumenFree = northTyumenFreeStock(tyumenStock, tyumenInTransit, tyumenPlannedOrder, tyumenTarget, tyumen?.warehouseStock, tyumen?.warehouseTransit);
  const allocation = allocateNorthNeedByCity(total, tyumenFree);
  const supplierUnitSize = Number(position.supplierUnitSize || 1);
  const supplierParts = [];
  if (tyumenSupplierNeed > 0) supplierParts.push({ key: "tyumen", label: "Тюмень", quantity: Number(tyumenSupplierNeed.toFixed(2)) });
  supplierParts.push(...allocation.supplierParts);
  const northNeed = Array.from(total.cities.entries())
    .filter(([cityKey]) => cityKey !== "tyumen")
    .reduce((sum, [, quantity]) => sum + quantity, 0);
  const supplierDemandNeed = Number((tyumenSupplierNeed + allocation.supplierNorthNeed).toFixed(2));
  const supplierNeed = supplierUnitsFromPieces(supplierDemandNeed, supplierUnitSize) || 0;
  const actualSupplierOrder = defaultNorthActualSupplierOrder(summary, position, supplierNeed);

  return {
    key: total.key,
    baseKey: total.baseKey || position.baseKey || total.key,
    variant: total.variant || position.variant || "",
    variantLabel: total.variantLabel || position.variantLabel || "",
    articleRaw: total.articleRaw || position.articleRaw || "",
    name: total.name || position.name,
    unit: total.unit || position.unit || "шт",
    supplierUnitSize,
    totalQuantity: Number(total.totalQuantity.toFixed(2)),
    blankBoxSize: position.blankBoxSize,
    cities: northTotalCityParts(total),
    northNeed: Number(northNeed.toFixed(2)),
    budget: { ...tyumen?.budget, prices: position.budgetPrices || [], line: position.christinaLine,
      preserveBlankPrices: position.preserveBlankPrices },
    tyumenOrder: tyumenUploadedOrder,
    tyumenPlannedOrder: Number(tyumenPlannedOrder.toFixed(2)),
    tyumenStock,
    tyumenInTransit,
    tyumenWarehouseStock: tyumen?.warehouseStock ?? null,
    tyumenWarehouseTransit: tyumen?.warehouseTransit ?? null,
    tyumenTarget,
    tyumenFree: Number(tyumenFree.toFixed(2)),
      fromTyumen: Number(allocation.fromTyumen.toFixed(2)),
      supplierNorthNeed: Number(allocation.supplierNorthNeed.toFixed(2)),
      supplierDemandNeed,
      supplierNeed,
    actualSupplierOrder,
    allowsRoundedShortSupplierOrder: summary.kind === "klapp",
    minimumExtra: actualSupplierOrder != null ? Number(Math.max(0, actualSupplierOrder - supplierNeed).toFixed(2)) : 0,
    supplierParts,
    tyumenParts: allocation.tyumenParts,
    hasTyumenSource: Boolean(tyumen),
    novacutanMinimum: position.novacutanMinimum ?? null,
  };
}

export function buildNorthPlan(summary, totals, tyumenSourceWorkbook = null) {
  const availability = readNorthTyumenAvailability(tyumenSourceWorkbook, summary.kind);
  const positionByKey = new Map(summary.positions.map((position) => [position.key, position]));
  const planRows = [];

  for (const total of totals.values()) {
    const position = positionByKey.get(total.key) || {
      key: total.key,
      name: total.name,
      articleRaw: total.articleRaw,
      unit: total.unit,
    };
    const tyumen = northAvailabilityForTotal(availability, total, summary.kind);
    const tyumenTableOrder = tyumen
      ? (tyumen.hasOrderedFact ? (tyumen.orderedFact ?? 0) : roundHalfUp(Math.max(0, tyumen.recommended || 0)))
      : 0;
    planRows.push(northPlanRowFromTotal(summary, total, position, tyumen, tyumenTableOrder));
  }

  return planRows.sort((left, right) => {
    const supplierDiff = Number(right.supplierNeed > 0) - Number(left.supplierNeed > 0);
    if (supplierDiff) return supplierDiff;
    return left.name.localeCompare(right.name, "ru");
  });
}

export function editedNorthTotals(result, editsByKey) {
  const totals = new Map();
  for (const row of result.planRows) {
    const edit = editsByKey.get(row.key);
    const cityQuantities = edit?.cities || Object.fromEntries((row.cities || []).map((city) => [city.key, city.quantity]));
    const cities = new Map();
    let totalQuantity = 0;
    for (const city of NORTH_CITIES) {
      const quantity = parseNumber(cityQuantities[city.key]) ?? 0;
      if (quantity <= 0) continue;
      cities.set(city.key, quantity);
      totalQuantity += quantity;
    }
    totals.set(row.key, {
      key: row.key,
      baseKey: row.baseKey || row.key,
      variant: row.variant || "",
      variantLabel: row.variantLabel || "",
      articleRaw: row.articleRaw,
      name: row.name,
      unit: row.unit,
      totalQuantity,
      cities,
    });
  }
  return totals;
}

export function recalculateEditedNorthPlan(result, totals) {
  return result.planRows.map((row) => {
    const total = totals.get(row.key) || {
      key: row.key,
      articleRaw: row.articleRaw,
      name: row.name,
      unit: row.unit,
      totalQuantity: 0,
      cities: new Map(),
    };
    const tyumen = row.hasTyumenSource
      ? {
          stock: row.tyumenStock,
          inTransit: row.tyumenInTransit,
          targetStock: row.tyumenTarget,
          warehouseStock: row.tyumenWarehouseStock,
          warehouseTransit: row.tyumenWarehouseTransit,
        }
      : null;
    return { ...northPlanRowFromTotal(result.summary, total, row, tyumen, row.tyumenPlannedOrder), budgetComment: row.budgetComment };
  });
}

export function actualQuantityByKey(planRows, editsByKey, options = {}) {
  const quantities = new Map();
  for (const row of planRows) {
    const edited = editsByKey.has(row.key) ? parseNumber(editsByKey.get(row.key).actualSupplierOrder) : null;
    const value = edited ?? row.actualSupplierOrder ?? 0;
    if (!options.allowShortSupplierOrder && Number(value || 0) < Number(row.supplierNeed || 0)) {
      throw new Error(`По позиции «${row.name}» факт у поставщика меньше общей нехватки Тюмени и северных городов.`);
    }
    quantities.set(row.key, Number(value || 0));
  }
  return quantities;
}
