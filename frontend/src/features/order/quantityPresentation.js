import { normalizeOrderValue } from "./editRules.js";

export function quantityDivergesFromRecommendation(row, editValue) {
  const recommended = Number(row.recommended);
  if (!Number.isFinite(recommended)) return false;
  let value;
  try {
    value = normalizeOrderValue(editValue ?? row.inserted);
  } catch {
    return true;
  }
  if (value == null) return recommended >= 1.5;
  return value !== recommended;
}

export function roundingComment(row) {
  return String(row.autoComment || "").trim();
}
