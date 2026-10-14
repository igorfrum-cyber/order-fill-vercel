import { normalizeArticle, volumeAwareSimilarity } from "./../../domain/matching/identity.js";
import { novacutanNameScore, novacutanPositionKey } from "./../../domain/matching/novacutan.js";
import { northNameKey } from "./../../domain/north/comments.js";
import { sheetBounds, sheetCellValue } from "./../../infrastructure/excel/xml.js";
import { asText, parseNumber } from "./../../shared/text.js";
import { detectCalculationColumns, detectColumns } from "./../../infrastructure/excel/columns.js";
import { isSourceTotalRow } from "./../../infrastructure/excel/source.js";
import { tyumenColumns } from "./../tyumen/merge.js";
import { budgetSourceMetrics } from "./../../infrastructure/excel/pricing.js";

export function sourceKeyCandidatesForNorth(item, kind) {
  const keys = new Set();
  if (kind === "novacutan") keys.add(novacutanPositionKey(item.name));
  if (!item.articleRaw) keys.add(northNameKey(item.name));
  const preserved = normalizeArticle(item.articleRaw, { preserveHyphen: true });
  const regular = normalizeArticle(item.articleRaw, { preserveHyphen: false });
  if (preserved) keys.add(preserved);
  if (regular) keys.add(regular);
  return Array.from(keys).filter(Boolean);
}

export function readNorthTyumenAvailability(workbook, kind) {
  if (!workbook) return new Map();
  const detection = detectColumns(workbook, "source");
  const calculationColumns = detectCalculationColumns(detection, { required: false });
  const { maxRow, maxColumn } = sheetBounds(detection.sheet);
  const availability = new Map();
  const locationColumns = tyumenColumns(detection);

  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    if (isSourceTotalRow(detection, row, maxColumn)) continue;
    const articleRaw = asText(sheetCellValue(detection.sheet, row, detection.columns.article));
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    if (!articleRaw && !name) continue;

    const stock = parseNumber(sheetCellValue(detection.sheet, row, detection.columns.stock)) ?? 0;
    const inTransit = parseNumber(sheetCellValue(detection.sheet, row, detection.columns.inTransit)) ?? 0;
    const recommended = parseNumber(sheetCellValue(detection.sheet, row, detection.columns.recommended)) ?? 0;
    const orderedFactRaw = sheetCellValue(detection.sheet, row, detection.columns.orderedFact);
    const orderedFact = parseNumber(orderedFactRaw);
    const hasOrderedFact = asText(orderedFactRaw) !== "";
    const targetStock = calculationColumns?.targetStock
      ? parseNumber(sheetCellValue(detection.sheet, row, calculationColumns.targetStock))
      : null;
    const item = {
      row,
      articleRaw,
      name,
      stock,
      inTransit,
      warehouseStock: locationColumns.warehouseStock ? parseNumber(sheetCellValue(detection.sheet, row, locationColumns.warehouseStock)) ?? 0 : null,
      warehouseTransit: locationColumns.warehouseTransit ? parseNumber(sheetCellValue(detection.sheet, row, locationColumns.warehouseTransit)) ?? 0 : null,
      recommended,
      hasOrderedFact,
      orderedFact: orderedFact ?? null,
      targetStock: targetStock ?? Math.max(0, stock + inTransit + recommended),
      budget: budgetSourceMetrics(detection, row),
    };

    for (const key of sourceKeyCandidatesForNorth(item, kind)) {
      if (!availability.has(key)) availability.set(key, item);
    }
  }

  return availability;
}

export function northAvailabilityForTotal(availability, total, kind) {
  const exact = availability.get(total.key);
  if (exact) return exact;
  const baseExact = total.baseKey ? availability.get(total.baseKey) : null;
  if (baseExact) return baseExact;
  if (!total.articleRaw) {
    const scored = Array.from(new Set(availability.values()))
      .map((item) => ({ item, score: volumeAwareSimilarity(total.name, item.name, total.unit) }))
      .sort((left, right) => right.score - left.score);
    if (!scored.length || scored[0].score < 0.9) return null;
    if (scored.length > 1 && scored[0].score - scored[1].score < 0.05) return null;
    return scored[0].item;
  }
  if (kind !== "novacutan") return null;

  const scored = Array.from(new Set(availability.values()))
    .map((item) => ({ item, score: novacutanNameScore(total.name, item.name, total.unit) }))
    .sort((left, right) => right.score - left.score);
  if (!scored.length || scored[0].score < 0.72) return null;
  if (scored[0].score >= 0.9) return scored[0].item;
  if (scored.length > 1 && scored[0].score - scored[1].score < 0.08) return null;
  return scored[0].item;
}
