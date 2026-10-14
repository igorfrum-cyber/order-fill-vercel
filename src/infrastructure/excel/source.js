import { cellKey, columnNumberToName, elements, firstElement, setNumericCell, setTextCell, sheetBounds, sheetCellValue } from "./xml.js";
import { asText, normalizeHeader, parseNumber } from "./../../shared/text.js";
import { expectedPeriods, formatPeriod, parsePeriodRange, rangesEqual } from "./../../domain/calculation/periods.js";
import { calculateTargetNew, categoryCoefficient, categoryFromCumulative, newProductCalculation, normalizeCategory } from "./../../domain/calculation/formulas.js";
import { canonicalProductArticle, chzNameSimilarity, isChzCloneName, isCombinedChzName, levissimeRussianIdentity, levissimeVolumeSignature, matchingLevissimeEnglishNames, normalizeArticle } from "./../../domain/matching/identity.js";
import { articleNormalizeOptions } from "./../../domain/brands/rules.js";
import { detectCalculationColumns } from "./columns.js";

export function findSourcePeriods(workbook) {
  let main = null;
  let previous = null;
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 40); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const text = asText(sheetCellValue(sheet, row, col));
        if (!text) continue;
        const parsed = parsePeriodRange(text);
        if (!parsed) continue;
        const normalized = normalizeHeader(text);
        if (normalized.includes("прошлый период")) previous = parsed;
        else if (normalized.includes("период")) main = parsed;
      }
      if (main && previous) return { main, previous };
    }
  }
  return { main, previous };
}

export function validateSourcePeriods(workbook, orderMonth) {
  const expected = expectedPeriods(orderMonth);
  const actual = findSourcePeriods(workbook);
  if (!actual.main || !actual.previous) throw new Error("Не нашел в таблице параметры периода и прошлого периода. Проверьте выгрузку из 1С.");
  if (!rangesEqual(actual.main, expected.main) || !rangesEqual(actual.previous, expected.previous)) {
    throw new Error(`Таблица расчета заказа сформирована не за тот период. Для заказа на ${expected.label} нужен период ${formatPeriod(expected.main)}, прошлый период ${formatPeriod(expected.previous)}. В загруженной таблице: период ${formatPeriod(actual.main)}, прошлый период ${formatPeriod(actual.previous)}. Переделайте выгрузку из 1С с правильными параметрами.`);
  }
  return {
    orderMonthLabel: expected.label,
    expectedMainPeriod: formatPeriod(expected.main),
    expectedPreviousPeriod: formatPeriod(expected.previous),
    actualMainPeriod: formatPeriod(actual.main),
    actualPreviousPeriod: formatPeriod(actual.previous),
  };
}

export function isUrengoySource(workbook) {
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 40); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const text = normalizeHeader(sheetCellValue(sheet, row, col));
        if (text.includes("уренгой") || text.includes("новый уренгой")) return true;
      }
    }
  }
  return false;
}

export function detectDeliveryWeeks(workbook) {
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 40); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const text = asText(sheetCellValue(sheet, row, col));
        const normalized = normalizeHeader(text);
        if (!normalized.includes("срок") || !normalized.includes("постав")) continue;
        const weeks = parseNumber(text);
        if (weeks != null && weeks > 0) return weeks;
      }
    }
  }
  return null;
}

export function maxMonthlySales(sheet, row, salesColumns) {
  let max = 0;
  for (const col of salesColumns) {
    const value = parseNumber(sheetCellValue(sheet, row, col));
    if (value != null) max = Math.max(max, value);
  }
  return max;
}

export function calculateUrengoyRecommended(sheet, row, urengoyInfo) {
  const maxSales = maxMonthlySales(sheet, row, urengoyInfo.salesColumns);
  const category = sheetCellValue(sheet, row, urengoyInfo.categoryColumn);
  const categoryPart = maxSales * categoryCoefficient(category, urengoyInfo.rule);
  const deliveryPart = maxSales * urengoyInfo.deliveryCoefficient;
  return {
    value: Number((categoryPart + deliveryPart).toFixed(2)),
    maxSales,
    category: normalizeCategory(category),
  };
}

export function isSourceTotalRow(detection, row, maxColumn) {
  const summaryAreaEnd = Math.min(maxColumn, Math.max(1, detection.columns.name - 1));
  for (let col = 1; col <= summaryAreaEnd; col += 1) {
    const text = normalizeHeader(sheetCellValue(detection.sheet, row, col));
    if (text === "итого" || text === "total") return true;
  }
  return false;
}

export function readSourceRows(detection, maxRow, maxColumn, rule) {
  const rows = [];
  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    if (isSourceTotalRow(detection, row, maxColumn)) continue;
    const articleRaw = asText(sheetCellValue(detection.sheet, row, detection.columns.article));
    const article = normalizeArticle(articleRaw, articleNormalizeOptions(rule));
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    if (!article && !name) continue;
    rows.push({ row, articleRaw, article, name, isChz: isChzCloneName(name) });
  }
  return rows;
}

export function sumColumns(sheet, targetRow, sourceRowsList, columns) {
  for (const col of columns.filter(Boolean)) {
    const total = sourceRowsList.reduce((sum, row) => sum + (parseNumber(sheetCellValue(sheet, row, col)) ?? 0), 0);
    setNumericCell(sheet, targetRow, col, Number(total.toFixed(2)));
  }
}

export function mergeFactAndComment(sheet, sourceRowsList, columns) {
  const facts = sourceRowsList.map((row) => parseNumber(sheetCellValue(sheet, row, columns.orderedFact))).filter((value) => value != null);
  const comments = sourceRowsList.map((row) => asText(sheetCellValue(sheet, row, columns.comment))).filter(Boolean);
  return {
    fact: facts.length ? Number(facts.reduce((sum, value) => sum + value, 0).toFixed(2)) : null,
    comment: Array.from(new Set(comments)).join("; "),
  };
}

export function removeWorksheetRows(sheet, rowNumbers) {
  const rowsToDelete = Array.from(new Set(rowNumbers)).sort((left, right) => left - right);
  if (!rowsToDelete.length) return;
  const rowsToDeleteSet = new Set(rowsToDelete);
  const sheetData = firstElement(sheet.xml, "sheetData");

  for (const rowNode of elements(sheetData, "row")) {
    const rowNumber = Number(rowNode.getAttribute("r"));
    if (rowsToDeleteSet.has(rowNumber)) rowNode.parentNode?.removeChild(rowNode);
  }

  for (const rowNode of elements(sheetData, "row")) {
    const originalRow = Number(rowNode.getAttribute("r"));
    const deletedBefore = rowsToDelete.filter((row) => row < originalRow).length;
    if (deletedBefore) rowNode.setAttribute("r", String(originalRow - deletedBefore));
  }

  const nextCells = new Map();
  for (const cell of sheet.cells.values()) {
    if (rowsToDeleteSet.has(cell.row)) continue;
    const deletedBefore = rowsToDelete.filter((row) => row < cell.row).length;
    if (deletedBefore) {
      cell.row -= deletedBefore;
      cell.ref = `${columnNumberToName(cell.col)}${cell.row}`;
      cell.node.setAttribute("r", cell.ref);
    }
    nextCells.set(cellKey(cell.row, cell.col), cell);
  }
  sheet.cells = nextCells;
}

export function monthlyValues(sheet, row, salesColumns) {
  return salesColumns.map((col) => parseNumber(sheetCellValue(sheet, row, col)));
}

export function recalculateSourceTable(detection, deliveryWeeks, rule, calculationColumns = detectCalculationColumns(detection)) {
  const { sheet, columns } = detection;
  const { maxRow, maxColumn } = sheetBounds(sheet);
  const rows = readSourceRows(detection, maxRow, maxColumn, rule);
  const safeDeliveryWeeks = Math.max(1, deliveryWeeks || 1);
  const deliveryCoefficient = 0.25 * safeDeliveryWeeks;
  let totalRevenue = 0;

  const calculated = rows.map((item) => {
    const values = monthlyValues(sheet, item.row, calculationColumns.salesColumns);
    const totalQuantity = values.reduce((sum, value) => sum + (value ?? 0), 0);
    const revenue = parseNumber(sheetCellValue(sheet, item.row, calculationColumns.revenue)) ?? 0;
    totalRevenue += revenue;
    return { ...item, values, totalQuantity, revenue };
  });

  const sorted = [...calculated].sort((left, right) => right.revenue - left.revenue);
  let cumulativeRevenue = 0;
  const rankMetrics = new Map();
  for (const item of sorted) {
    cumulativeRevenue += item.revenue;
    const revenuePercent = totalRevenue > 0 ? (item.revenue / totalRevenue) * 100 : 0;
    const cumulativePercent = totalRevenue > 0 ? (cumulativeRevenue / totalRevenue) * 100 : 100;
    rankMetrics.set(item.row, {
      revenuePercent,
      cumulativePercent,
      category: categoryFromCumulative(cumulativePercent),
    });
  }

  for (const item of calculated) {
    const metrics = rankMetrics.get(item.row) || { revenuePercent: 0, cumulativePercent: 100, category: "C" };
    const novelty = newProductCalculation(item.values, safeDeliveryWeeks);
    const monthlyNeed = novelty?.monthlyNeed ?? calculateTargetNew(item.values);
    const category = novelty ? `${metrics.category}/New` : metrics.category;
    const targetStock = novelty
      ? novelty.targetStock
      : monthlyNeed * categoryCoefficient(metrics.category, rule) + monthlyNeed * deliveryCoefficient;
    const stock = parseNumber(sheetCellValue(sheet, item.row, columns.stock)) ?? 0;
    const inTransit = parseNumber(sheetCellValue(sheet, item.row, columns.inTransit)) ?? 0;
    const recommended = Math.max(0, targetStock - stock - inTransit);

    setNumericCell(sheet, item.row, calculationColumns.totalQuantity, Number(item.totalQuantity.toFixed(2)));
    setNumericCell(sheet, item.row, calculationColumns.revenuePercent, Number(metrics.revenuePercent.toFixed(2)));
    setNumericCell(sheet, item.row, calculationColumns.cumulativePercent, Number(metrics.cumulativePercent.toFixed(2)));
    setTextCell(sheet, item.row, calculationColumns.category, category);
    setNumericCell(sheet, item.row, calculationColumns.averageMonthly, Number((item.totalQuantity / calculationColumns.salesColumns.length).toFixed(2)));
    setNumericCell(sheet, item.row, calculationColumns.targetStock, Number(targetStock.toFixed(2)));
    setNumericCell(sheet, item.row, columns.recommended, Number(recommended.toFixed(2)));
  }
}

export function rebuildSourceWithChz(detection, deliveryWeeks, rule, calculationColumns = detectCalculationColumns(detection)) {
  const { sheet, columns } = detection;
  const { maxRow, maxColumn } = sheetBounds(sheet);
  const rows = readSourceRows(detection, maxRow, maxColumn, rule);
  const rowsByArticle = new Map();
  for (const row of rows) {
    if (asText(sheetCellValue(sheet,row.row,columns.comment)).includes('Проверить: неоднозначная позиция.')) continue;
    if (!row.article) continue;
    const key = canonicalProductArticle(row.article, rule);
    if (!rowsByArticle.has(key)) rowsByArticle.set(key, []);
    rowsByArticle.get(key).push(row);
  }

  const rowsToDelete = [];
  const consumed = new Set();
  for (const group of rowsByArticle.values()) {
    const normalRows = group.filter((row) => !row.isChz);
    const chzRows = group.filter((row) => row.isChz);
    for (const target of normalRows) {
      if (consumed.has(target.row)) continue;
      const identity = rule.label === "LeviSsime" ? levissimeRussianIdentity(target.name, rule) : null;
      const matchingNormalRows = identity ? normalRows.filter(row => row !== target && !consumed.has(row.row)
        && levissimeRussianIdentity(row.name, rule) === identity) : [];
      const matchingChzRows = chzRows.filter((row) => !consumed.has(row.row)
        && (rule.label !== "LeviSsime" || levissimeVolumeSignature(target.name) === levissimeVolumeSignature(row.name))
        && ((identity && levissimeRussianIdentity(row.name, rule) === identity)
          || (rule.label === "LeviSsime" && matchingLevissimeEnglishNames(target.name, row.name))
          || chzNameSimilarity(target.name, row.name, rule) >= 0.9));
      const matchingRows = [...matchingNormalRows, ...matchingChzRows];
      if (!matchingRows.length) continue;

      const sourceRowsList = [target, ...matchingRows].map((row) => row.row);
      const hasChz = matchingChzRows.length || [target, ...matchingNormalRows].some(row => isCombinedChzName(row.name));
      const mergedName = hasChz ? `ЧЗ + ${target.name.replace(/^чз\s*\+\s*/iu, "")}` : target.name;
      setTextCell(sheet, target.row, columns.name, mergedName);
      if (columns.name !== 1 && columns.article !== 1) setTextCell(sheet, target.row, 1, mergedName);
      sumColumns(sheet, target.row, sourceRowsList, [
        ...calculationColumns.salesColumns,
        calculationColumns.totalQuantity,
        calculationColumns.revenue,
        calculationColumns.previousQuantity,
        columns.stock,
        columns.inTransit,
      ]);
      const fact = mergeFactAndComment(sheet, sourceRowsList, columns);
      setNumericCell(sheet, target.row, columns.orderedFact, fact.fact);
      setTextCell(sheet, target.row, columns.comment, fact.comment);
      for (const row of matchingRows) {
        consumed.add(row.row);
        rowsToDelete.push(row.row);
      }
    }
  }

  removeWorksheetRows(sheet, rowsToDelete);
  if (calculationColumns.salesColumns.length) recalculateSourceTable(detection, deliveryWeeks, rule, calculationColumns);
}
