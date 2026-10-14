import { ARTICLE_TRANSLATION } from "./../matching/identity.js";
import { asText } from "./../../shared/text.js";

export function roundHalfUp(value) {
  return Math.floor(value + 0.5);
}

export function normalizeCategory(value) {
  return asText(value)
    .replace(/[АВСавс]/g, (ch) => ARTICLE_TRANSLATION.get(ch) || ch)
    .toUpperCase()
    .replace(/\s+/g, "");
}

export function categoryCoefficient(value, rule = null) {
  const category = normalizeCategory(value);
  if (rule?.label === "NOVACUTAN") {
    if (category === "C") return 1.5;
    if (category === "A+" || category === "A" || category === "B") return 2;
  }
  if (category === "A+") return 2;
  if (category === "A") return 1.75;
  if (category === "B") return 1.5;
  if (category === "C") return 1;
  return 1;
}

export function categoryFromCumulative(percent) {
  if (percent <= 50) return "A+";
  if (percent <= 80) return "A";
  if (percent <= 95) return "B";
  return "C";
}

export function calculateTargetNew(values) {
  const numeric = values.map((value) => (value == null ? null : Number(value)));
  const total = numeric.reduce((sum, value) => sum + (Number.isFinite(value) ? value : 0), 0);
  const threshold = total / 24;
  const stable = numeric.every((value) => Number.isFinite(value) && value > 0) && Math.min(...numeric) > threshold;
  if (stable) {
    const average = total / numeric.length;
    const secondMonth = numeric[1] ?? 0;
    const firstThreeAverage = numeric.slice(0, 3).reduce((sum, value) => sum + value, 0) / 3;
    return (average + secondMonth + firstThreeAverage) / 3;
  }

  const filtered = numeric.filter((value) => Number.isFinite(value) && value > 0 && value > threshold);
  if (!filtered.length) return 0;
  return filtered.reduce((sum, value) => sum + value, 0) / filtered.length;
}

export function positiveSale(value) {
  return Number.isFinite(value) && value > 0;
}

export function newProductCalculation(values, deliveryWeeks) {
  const numeric = values.map((value) => (value == null ? null : Number(value)));
  let suffixLength = 0;
  for (let index = numeric.length - 1; index >= 0; index -= 1) {
    if (!positiveSale(numeric[index])) break;
    suffixLength += 1;
  }
  if (suffixLength < 1 || suffixLength > 3) return null;

  const firstNoveltyIndex = numeric.length - suffixLength;
  const hadPreviousSales = numeric.slice(0, firstNoveltyIndex).some((value) => positiveSale(value));
  if (hadPreviousSales) return null;

  const noveltyValues = numeric.slice(firstNoveltyIndex).filter(positiveSale);
  const maxMonth = Math.max(...noveltyValues);
  if (!Number.isFinite(maxMonth) || maxMonth <= 0) return null;

  const deliveryCoefficient = 0.25 * Math.max(1, deliveryWeeks || 1);
  return {
    months: suffixLength,
    maxMonth,
    monthlyNeed: maxMonth * 1.5,
    targetStock: maxMonth * 1.5 + maxMonth * deliveryCoefficient,
  };
}
