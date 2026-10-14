import { MONTHS_RU } from "./../../domain/calculation/periods.js";
import { normalizeHeader } from "./../../shared/text.js";
import { sheetBounds, sheetCellValue } from "./xml.js";

export function sourceMatchers() {
  return {
    article: (h) => h.includes("арт") || h.includes("артикул") || h.includes("код"),
    name: (h) => h.includes("товар") || h.includes("номенклатура") || h.includes("наименование") || h.includes("название"),
    recommended: (h) => h.includes("рекоменд") && h.includes("заказ"),
    stock: (h) => h.includes("остаток"),
    inTransit: (h) => h.includes("в пути"),
    orderedFact: (h) => h.includes("заказано") && h.includes("факт"),
    comment: (h) => h.includes("комментар"),
  };
}

export function isMonthHeader(value) {
  const normalized = normalizeHeader(value);
  if (!/\b20\d{2}\b/.test(normalized)) return false;
  return MONTHS_RU.slice(1).some((month) => normalized.includes(month));
}

export function detectUrengoyColumns(detection) {
  const { sheet, headerRow, columns } = detection;
  const { maxColumn } = sheetBounds(sheet);
  let category = null;
  const salesColumns = [];
  for (let col = 1; col <= maxColumn; col += 1) {
    const currentHeader = normalizeHeader(sheetCellValue(sheet, headerRow, col));
    const upperHeader = sheetCellValue(sheet, headerRow - 1, col);
    if (!category && currentHeader === "категория") category = col;
    if (col < columns.recommended && currentHeader === "количество" && isMonthHeader(upperHeader)) {
      salesColumns.push(col);
    }
  }
  if (!category) throw new Error("Для Уренгоя не нашел колонку «Категория» в таблице заказа.");
  if (!salesColumns.length) throw new Error("Для Уренгоя не нашел месячные колонки продаж с заголовком «Количество».");
  return { category, salesColumns };
}

export function detectCalculationColumns(detection, options = {}) {
  const { sheet, headerRow, columns } = detection;
  const { maxColumn } = sheetBounds(sheet);
  const found = {
    salesColumns: [],
    totalQuantity: null,
    revenue: null,
    revenuePercent: null,
    cumulativePercent: null,
    category: null,
    averageMonthly: null,
    previousQuantity: null,
    targetStock: null,
  };

  for (let col = 1; col <= maxColumn; col += 1) {
    const currentHeader = normalizeHeader(sheetCellValue(sheet, headerRow, col));
    const upperHeader = sheetCellValue(sheet, headerRow - 1, col);
    const upperNormalized = normalizeHeader(upperHeader);
    if (!currentHeader) continue;

    if (col < columns.recommended && currentHeader === "количество" && isMonthHeader(upperHeader)) found.salesColumns.push(col);
    else if (!found.totalQuantity && col < columns.recommended && currentHeader === "количество" && upperNormalized.includes("итого")) found.totalQuantity = col;
    else if (!found.revenue && currentHeader.includes("сумма") && currentHeader.includes("выруч")) found.revenue = col;
    else if (!found.revenuePercent && currentHeader.includes("%") && currentHeader.includes("выруч")) found.revenuePercent = col;
    else if (!found.cumulativePercent && currentHeader.includes("кумулятив")) found.cumulativePercent = col;
    else if (!found.category && currentHeader === "категория") found.category = col;
    else if (!found.averageMonthly && currentHeader.includes("среднее") && currentHeader.includes("месяц")) found.averageMonthly = col;
    else if (!found.previousQuantity && currentHeader.includes("количество") && currentHeader.includes("прошлый")) found.previousQuantity = col;
    else if (!found.targetStock && currentHeader.includes("целевой") && currentHeader.includes("запас")) found.targetStock = col;
  }

  const missing = [];
  if (!found.salesColumns.length && !options.allowNoSales) missing.push("месячные продажи");
  for (const key of ["totalQuantity", "revenue", "revenuePercent", "cumulativePercent", "category", "averageMonthly", "previousQuantity", "targetStock"]) {
    if (!found[key]) missing.push(key);
  }
  if (missing.length && options.required !== false) throw new Error(`Не нашел расчетные колонки: ${missing.join(", ")}.`);
  if (missing.length) return null;
  return found;
}

export function blankMatchers(options = {}) {
  const isQuantityHeader = (h) => h.includes("кол во") || h.includes("количество") || h.includes("кол-во") || h.includes("к во") || h.includes("qty");
  const isPackageHeader = (h) => h.includes("короб") || h.includes("упак") || h.split(" ").includes("уп");
  const quantityMatcher = options.quantityHeader === "order"
    ? (h) => h === "заказ" || h.includes("коробка заказ")
    : options.quantityHeader === "exactQuantity"
      ? (h) => h === "количество"
    : options.quantityHeader === "anyOrder"
      ? (h) => h === "заказ" || h.includes("коробка заказ") || (isQuantityHeader(h) && !isPackageHeader(h))
      : isQuantityHeader;
  const boxMatcher = options.boxHeader === "packageQuantity"
    ? (h) => h.includes("кол во в уп") || h === "кол во" || h === "количество"
    : (h) => h.includes("короб") || (h.includes("шт") && h.includes("упак"));
  const matchers = {
    article: (h) => h.includes("арт") || h.includes("артикул") || h.includes("код"),
    name: (h) => h.includes("товар") || h.includes("номенклатура") || h.includes("наименование") || h.includes("название") || h.includes("продукт"),
    unit: (h) => h.includes("объем") || h.includes("обьем") || h.includes("форма выпуска") || (h.includes("мл") && h.includes("гр")),
    quantity: quantityMatcher,
  };
  if (options.requireArticle === false) {
    delete matchers.article;
  }
  if (options.requireBox !== false) {
    matchers.boxSize = boxMatcher;
  }
  if (options.requireUnit === false) {
    delete matchers.unit;
  }
  return matchers;
}

export function combinations(arrays) {
  return arrays.reduce((acc, current) => acc.flatMap((items) => current.map((item) => [...items, item])), [[]]);
}

export function detectColumns(workbook, mode, options = {}) {
  const matchers = mode === "source" ? sourceMatchers() : blankMatchers(options);
  const required = Object.keys(matchers);
  let bestFound = {};
  let bestScore = -1;
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 120); row += 1) {
      const candidates = Object.fromEntries(required.map((key) => [key, []]));
      for (let col = 1; col <= maxColumn; col += 1) {
        const header = normalizeHeader(sheetCellValue(sheet, row, col));
        if (!header) continue;
        for (const key of required) if (matchers[key](header)) candidates[key].push(col);
      }
      const foundKeys = required.filter((key) => candidates[key].length > 0);
      if (foundKeys.length > bestScore) {
        bestScore = foundKeys.length;
        bestFound = Object.fromEntries(foundKeys.map((key) => [key, candidates[key][0]]));
      }
      if (foundKeys.length !== required.length) continue;
      let bestForRow = null;
      for (const combo of combinations(required.map((key) => candidates[key]))) {
        if (new Set(combo).size !== combo.length) continue;
        const span = Math.max(...combo) - Math.min(...combo) + 1;
        if (!bestForRow || span < bestForRow.span) {
          bestForRow = { span, columns: Object.fromEntries(required.map((key, index) => [key, combo[index]])) };
        }
      }
      if (bestForRow) return { sheet, sheetName: sheet.name, headerRow: row, columns: bestForRow.columns };
    }
  }
  throw new Error(`Не удалось найти все нужные колонки: ${required.join(", ")}. Найдено: ${Object.keys(bestFound).join(", ") || "ничего"}.`);
}
