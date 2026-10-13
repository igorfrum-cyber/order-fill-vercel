import { normalizeHeader, parseNumber } from "./../../shared/text.js";
import { roundHalfUp } from "./../calculation/formulas.js";
import { brandRule } from "./../brands/rules.js";
import { novacutanMatchKey } from "./../matching/novacutan.js";

export function calculateAdjustedQuantity(recommended, rule, boxSizeValue) {
  const rounded = roundHalfUp(recommended);
  if (!rule.allowSmallPositiveOrder && recommended < 1.5) return { rounded, inserted: null, autoComment: "", boxAdjusted: false };
  if (rounded <= 0) return { rounded, inserted: null, autoComment: "", boxAdjusted: false };

  if (rule.adjustment === "none") {
    return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
  }

  if (rule.adjustment === "multiple") {
    return calculateMultipleAdjustedQuantity(rounded, rule.multiple, rule.adjustmentComment);
  }

  if (rule.adjustment === "nearestMultiple") {
    return calculateNearestMultipleAdjustedQuantity(rounded, rule.multiple, rule.adjustmentComment);
  }

  if (rule.adjustment === "minimum") {
    const minimum = Math.round(parseNumber(boxSizeValue) || 0);
    if (minimum > 0 && rounded < minimum) {
      return { rounded, inserted: minimum, autoComment: rule.adjustmentComment, boxAdjusted: true };
    }
    return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
  }

  const boxSize = parseNumber(boxSizeValue);
  if (!boxSize || boxSize <= 0) return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
  const box = Math.round(boxSize);
  return calculateMultipleAdjustedQuantity(rounded, box, rule.adjustmentComment);
}

export function adjustQuantityForBrand(recommended, brand = "angiopharm", boxSizeValue = null) {
  const rule = brandRule(brand);
  return calculateAdjustedQuantity(recommended, rule, boxSizeValue ?? rule.multiple);
}

export function orderForItem(item, rule, adjustmentValue) {
  if (!item.hasOrderedFact) return calculateAdjustedQuantity(item.recommended, rule, adjustmentValue);
  return {
    ...calculateAdjustedQuantity(item.orderedFact, rule, adjustmentValue),
    autoComment: "",
    fromOrderedFact: true,
  };
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

export function convertOrderToSupplierUnits(order, unitSize = 1) {
  const size = Number(unitSize || 1);
  if (!Number.isFinite(size) || size <= 1) return order;
  const rounded = supplierUnitsFromPieces(order.rounded, size);
  const inserted = supplierUnitsFromPieces(order.inserted, size);
  return {
    ...order,
    rounded,
    inserted,
    autoComment: inserted != null ? "коробки по 5 шт." : order.autoComment,
    supplierUnitSize: size,
  };
}

export function novacutanOrderForItem(item, rowInfo, rule) {
  return convertOrderToSupplierUnits(orderForItem(item, rule, rowInfo.blankBoxSize), rowInfo.supplierUnitSize);
}

export function calculateMultipleAdjustedQuantity(rounded, multiple, comment) {
  const step = Math.round(Number(multiple));
  if (step <= 0 || rounded % step === 0) return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };

  const lower = Math.floor(rounded / step) * step;
  const upper = Math.ceil(rounded / step) * step;
  const upPercent = (upper - rounded) / rounded;
  const downPercent = lower > 0 ? (rounded - lower) / rounded : Infinity;

  if (upPercent > 0 && upPercent <= 0.15) {
    return { rounded, inserted: upper, autoComment: comment, boxAdjusted: true };
  }
  if (downPercent > 0 && downPercent <= 0.05) {
    return { rounded, inserted: lower, autoComment: comment, boxAdjusted: true };
  }
  return { rounded, inserted: rounded, autoComment: "", boxAdjusted: false };
}

export function nearestMultipleValue(value, multiple) {
  const number = Number(value || 0);
  const step = Math.round(Number(multiple));
  if (!Number.isFinite(number) || number <= 0) return null;
  if (!Number.isFinite(step) || step <= 0) return Number(number.toFixed(2));
  const lower = Math.floor(number / step) * step;
  const upper = Math.ceil(number / step) * step;
  if (lower <= 0) return upper;
  const lowerDiff = number - lower;
  const upperDiff = upper - number;
  return upperDiff <= lowerDiff ? upper : lower;
}

export function calculateNearestMultipleAdjustedQuantity(rounded, multiple, comment) {
  const inserted = nearestMultipleValue(rounded, multiple);
  if (inserted == null) return { rounded, inserted: null, autoComment: "", boxAdjusted: false };
  return {
    rounded,
    inserted,
    autoComment: inserted !== rounded ? comment : "",
    boxAdjusted: inserted !== rounded,
  };
}

export function novacutanMinimumQuantity(name) {
  const text = normalizeHeader(name);
  if ((text.includes("mask") || text.includes("маск")) && (text.includes("filler") || text.includes("филлер"))) {
    return 10;
  }
  if (
    text.includes("fbio")
    || text.includes("bright")
    || text.includes("брайт")
    || text.includes("gentle")
    || text.includes("джентл")
    || text.includes("джентел")
  ) {
    return 50;
  }
  return 100;
}

export function novacutanSupplierUnitSize(name) {
  const key = novacutanMatchKey(name);
  return key === "mask-eye" || key === "mask-face" ? 5 : 1;
}

export function novacutanSummaryQuantity(position, total) {
  if (total <= 0) return null;
  const minimum = Number(position.novacutanMinimum || 100);
  return total < minimum ? minimum : Number(total.toFixed(2));
}

export function roundSupplierPack10(value) {
  if (value <= 0) return null;
  return Math.round(value / 10) * 10;
}

export function novacutanSupplierOrderQuantity(position, supplierNeed) {
  if (supplierNeed <= 0) return null;
  const minimum = Number(position.novacutanMinimum || 100);
  const base = Math.max(Number(supplierNeed), minimum);
  if (Number(position.supplierUnitSize || 1) > 1) return Number(base.toFixed(2));
  return roundSupplierPack10(base);
}
