import { setNumericCell, sheetBounds, sheetCellValue } from "./../../infrastructure/excel/xml.js";
import { asText, parseNumber } from "./../../shared/text.js";
import { normalizeArticle } from "./../../domain/matching/identity.js";
import { roundHalfUp } from "./../../domain/calculation/formulas.js";
import { calculateUrengoyRecommended, detectDeliveryWeeks, isSourceTotalRow, isUrengoySource, rebuildSourceWithChz, validateSourcePeriods } from "./../../infrastructure/excel/source.js";
import { detectCalculationColumns, detectColumns, detectUrengoyColumns } from "./../../infrastructure/excel/columns.js";
import { articleNormalizeOptions, brandRule } from "./../../domain/brands/rules.js";
import { budgetSourceMetrics } from "./../../infrastructure/excel/pricing.js";
import { detectSourceCity } from "./../../infrastructure/excel/cities.js";

export function readSource(workbook, orderMonth, rule = brandRule("angiopharm"), sourceFileName = "") {
  const periodInfo = validateSourcePeriods(workbook, orderMonth);
  const sourceCity = detectSourceCity(workbook, sourceFileName);
  const detection = detectColumns(workbook, "source");
  const isAngiopharm = rule.label === "ANGIOPHARM";
  const isUrengoy = isUrengoySource(workbook);
  const deliveryWeeks = Math.max(1, detectDeliveryWeeks(workbook) ?? 1);
  const calculationColumns = detectCalculationColumns(detection, { required: false });

  if (calculationColumns) {
    rebuildSourceWithChz(detection, deliveryWeeks, rule, calculationColumns);
  }

  const urengoyColumns = !calculationColumns && !isAngiopharm && isUrengoy ? detectUrengoyColumns(detection) : null;
  const urengoyInfo = !calculationColumns && !isAngiopharm && isUrengoy
    ? {
        rule,
        deliveryWeeks,
        deliveryCoefficient: 1 + 0.25 * deliveryWeeks,
        categoryColumn: urengoyColumns.category,
        salesColumns: urengoyColumns.salesColumns,
      }
    : null;
  const items = [];
  const { maxRow, maxColumn } = sheetBounds(detection.sheet);
  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    if (isSourceTotalRow(detection, row, maxColumn)) continue;
    const articleRaw = asText(sheetCellValue(detection.sheet, row, detection.columns.article));
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    const recommendedRaw = sheetCellValue(detection.sheet, row, detection.columns.recommended);
    const urengoyRecommended = urengoyInfo && (articleRaw || name || asText(recommendedRaw))
      ? calculateUrengoyRecommended(detection.sheet, row, urengoyInfo)
      : null;
    if (urengoyRecommended) setNumericCell(detection.sheet, row, detection.columns.recommended, urengoyRecommended.value);
    const recommendedValue = parseNumber(sheetCellValue(detection.sheet, row, detection.columns.recommended));
    const orderedFactRaw = sheetCellValue(detection.sheet, row, detection.columns.orderedFact);
    const orderedFact = parseNumber(orderedFactRaw);
    const hasOrderedFact = asText(orderedFactRaw) !== "";
    if (!articleRaw && !name && recommendedValue == null) continue;
    if (hasOrderedFact && orderedFact == null) throw new Error(`В строке ${row} таблицы заказа некорректно заполнено «Заказано по факту».`);
    const recommended = recommendedValue ?? 0;
    items.push({
      rowIndex: row,
      budget: budgetSourceMetrics(detection, row),
      articleRaw,
      article: normalizeArticle(articleRaw, articleNormalizeOptions(rule)),
      name,
      recommended,
      rounded: roundHalfUp(recommended),
      hasOrderedFact,
      orderedFact,
      sourceComment: asText(sheetCellValue(detection.sheet, row, detection.columns.comment)),
      stock: sheetCellValue(detection.sheet, row, detection.columns.stock),
      inTransit: sheetCellValue(detection.sheet, row, detection.columns.inTransit),
      urengoyRecommended: Boolean(urengoyRecommended),
      urengoyMaxSales: urengoyRecommended?.maxSales ?? null,
      urengoyCategory: urengoyRecommended?.category ?? "",
    });
  }
  return {
    workbook,
    detection,
    items,
    periodInfo: {
      ...periodInfo,
      sourceCity: sourceCity?.label || "",
      cityRule: isUrengoy ? "Новый Уренгой" : "",
      deliveryWeeks,
      deliveryCoefficient: calculationColumns ? 0.25 * deliveryWeeks : urengoyInfo?.deliveryCoefficient ?? null,
    },
  };
}
