import { columnNumberToName, firstElement, setNumericCell, setTextCell, sheetBounds, sheetCellValue } from "./../../infrastructure/excel/xml.js";
import { asText, normalizeHeader, parseNumber } from "./../../shared/text.js";
import { canonicalProductArticle, comparableChzName, levissimeRussianIdentity, matchingLevissimeEnglishNames, normalizeName } from "./../../domain/matching/identity.js";
import { brandRule } from "./../../domain/brands/rules.js";
import { novacutanPositionKey } from "./../../domain/matching/novacutan.js";
import { detectDeliveryWeeks, findSourcePeriods, isSourceTotalRow, readSourceRows, rebuildSourceWithChz, recalculateSourceTable, sumColumns } from "./../../infrastructure/excel/source.js";
import { detectCalculationColumns, detectColumns } from "./../../infrastructure/excel/columns.js";
import { loadXlsx, saveXlsx } from "./../../infrastructure/excel/archive.js";
import { validateNorthTyumenSourceWorkbook } from "./../../infrastructure/excel/cities.js";
import { roundHalfUp } from "./../../domain/calculation/formulas.js";
import { warehouseTransferQuantity } from "./../../domain/warehouse/stock.js";

export const TYUMEN_COLUMNS = {
  officeStock: "Офис: остаток",
  warehouseStock: "Склад: остаток",
  officeTransit: "Офис: в пути",
  warehouseTransit: "Склад: в пути",
};

export function tyumenColumns(detection) {
  const result = {};
  for (const [key, header] of Object.entries(TYUMEN_COLUMNS)) {
    for (let col = 1; col <= sheetBounds(detection.sheet).maxColumn; col += 1) {
      if (normalizeHeader(sheetCellValue(detection.sheet, detection.headerRow, col)) === normalizeHeader(header)) result[key] = col;
    }
  }
  return result;
}

export function tyumenProductKey(item, brand) {
  if (item.article) return `article:${canonicalProductArticle(item.article, brandRule(brand))}`;
  if (brand === "novacutan") return `name:${novacutanPositionKey(item.name)}`;
  return `name:${normalizeName(item.name.replace(/^\s*чз\s*\+?\s*/iu, ""))}`;
}

export function tyumenRows(workbook, brand, options = {}) {
  const detection = detectColumns(workbook, "source");
  if (Object.keys(tyumenColumns(detection)).length) throw new Error("Загружена уже объединённая таблица. Для двух складов нужны отдельные исходные таблицы офиса и склада.");
  const rule = brandRule(brand);
  const calculation = detectCalculationColumns(detection, options);
  rebuildSourceWithChz(detection, Math.max(1, detectDeliveryWeeks(workbook) ?? 1), rule, calculation);
  const bounds = sheetBounds(detection.sheet);
  const items = readSourceRows(detection, bounds.maxRow, bounds.maxColumn, rule);
  const byKey = new Map();
  for (const item of items) {
    if (!calculation.salesColumns.length && [calculation.totalQuantity, calculation.revenue].some((col) => (parseNumber(sheetCellValue(detection.sheet, item.row, col)) ?? 0) !== 0)) {
      throw new Error(`В отчёте склада нет месячных колонок, но у «${item.name}» указаны продажи или выручка. Нужен отчёт с разбивкой продаж по месяцам.`);
    }
    const key = tyumenProductKey(item, brand);
    // Preserve all unresolved duplicates for the existing report review UI.
    if (byKey.has(key)) {
      const previous=byKey.get(key);
      previous.ambiguous=true; item.ambiguous=true;
      byKey.set(`${key}:duplicate:${item.row}`,item);
    } else byKey.set(key, item);
  }
  return { detection, calculation, byKey };
}

export function compatibleTyumenNames(left, right, brand) {
  const clean = (name) => normalizeName(name.replace(/^\s*чз\s*\+?\s*/iu, ""));
  if (brand === "levissime" && left.article && right.article) {
    const rule = brandRule(brand);
    if (canonicalProductArticle(left.article, rule) === canonicalProductArticle(right.article, rule)) {
      const identity = levissimeRussianIdentity(left.name, rule);
      return (identity && identity === levissimeRussianIdentity(right.name, rule))
        || matchingLevissimeEnglishNames(left.name, right.name)
        || comparableChzName(left.name, rule) === comparableChzName(right.name, rule);
    }
  }
  return clean(left.name) === clean(right.name);
}

export function alignTyumenKeys(office, warehouse, brand) {
  // An ambiguous identity must not silently pick one warehouse counterpart.
  const conflicts=new Set();
  for(const side of [office,warehouse])for(const item of side.byKey.values())if(item.ambiguous)conflicts.add(tyumenProductKey(item,brand));
  for(const [key,a] of office.byKey){const b=warehouse.byKey.get(key);if(b&&!compatibleTyumenNames(a,b,brand))conflicts.add(tyumenProductKey(a,brand));}
  for(const [label,side] of [['office',office],['warehouse',warehouse]]) {
    for(const [key,item] of [...side.byKey])if(conflicts.has(tyumenProductKey(item,brand))) {
      item.ambiguous=true;side.byKey.delete(key);side.byKey.set(`review:${label}:${item.row}`,item);
    }
  }
  // Exact name fallback is allowed only when an article is missing on one side.
  // Distinct known articles must never be merged just because names coincide.
  for (const [key, item] of [...office.byKey]) {
    if(item.ambiguous)continue;
    if (warehouse.byKey.has(key)) continue;
    const candidates = [...warehouse.byKey].filter(([, other]) => !other.ambiguous && (!item.article || !other.article) && compatibleTyumenNames(item, other, brand));
    if (candidates.length > 1) throw new Error(`Неоднозначное соответствие по названию «${item.name}» между офисом и складом.`);
    if (candidates.length === 1) {
      const [targetKey] = candidates[0];
      if (office.byKey.has(targetKey)) throw new Error(`Несколько позиций офиса соответствуют одной позиции склада «${item.name}».`);
      office.byKey.delete(key);
      office.byKey.set(targetKey, item);
    }
  }
}

/**
 * Combine raw Tyumen office/warehouse history, then recompute ABC and demand.
 * Inputs are cloned; the returned workbook retains separate
 * location columns for subsequent North calculations. See docs/tyumen.md.
 * Ambiguous identities stay separate for report review; mismatched periods fail.
 */
export function mergeTyumenSources({ officeWorkbook, warehouseWorkbook, brand = "angiopharm", officeFileName = "", warehouseFileName = "", preserveFacts = false }) {
  validateNorthTyumenSourceWorkbook(officeWorkbook, officeFileName);
  validateNorthTyumenSourceWorkbook(warehouseWorkbook, warehouseFileName);
  const officeCopy = loadXlsx(saveXlsx(officeWorkbook));
  const warehouseCopy = loadXlsx(saveXlsx(warehouseWorkbook));
  const office = tyumenRows(officeCopy, brand);
  const warehouse = tyumenRows(warehouseCopy, brand, { allowNoSales: true });
  const warehouseHasSales = warehouse.calculation.salesColumns.length > 0;
  if (warehouseHasSales && JSON.stringify(findSourcePeriods(officeWorkbook)) !== JSON.stringify(findSourcePeriods(warehouseWorkbook))) {
    throw new Error("Периоды продаж в таблицах офиса и склада не совпадают.");
  }
  const deliveryWeeks = Math.max(1, detectDeliveryWeeks(officeWorkbook) ?? 1);
  if (warehouseHasSales && deliveryWeeks !== Math.max(1, detectDeliveryWeeks(warehouseWorkbook) ?? 1)) throw new Error("Срок поставки в таблицах офиса и склада различается. Укажите одинаковый срок поставки от поставщика.");
  // A new warehouse may have inventory but no monthly columns yet. Use the
  // office template so the combined report still has a complete sales timeline.
  const workbook = warehouseHasSales ? warehouseCopy : officeCopy;
  alignTyumenKeys(office, warehouse, brand);
  const monthHeaders = ({ detection, calculation }) => calculation.salesColumns.map((col) => normalizeHeader(sheetCellValue(detection.sheet, detection.headerRow - 1, col)));
  if (warehouseHasSales && JSON.stringify(monthHeaders(office)) !== JSON.stringify(monthHeaders(warehouse))) throw new Error("Месяцы продаж офиса и склада не совпадают.");
  const { detection, calculation } = warehouseHasSales ? warehouse : office;
  const { sheet, columns, headerRow } = detection;
  let lastRow = sheetBounds(sheet).maxRow;
  const firstExtra = sheetBounds(sheet).maxColumn + 1;
  const extras = Object.fromEntries(Object.entries(TYUMEN_COLUMNS).map(([key, label], index) => {
    setTextCell(sheet, headerRow, firstExtra + index, label);
    return [key, firstExtra + index];
  }));
  const keys = new Set([...warehouse.byKey.keys(), ...office.byKey.keys()]);
  for (const key of keys) {
    const a = office.byKey.get(key);
    const b = warehouse.byKey.get(key);
    if (a && b && !compatibleTyumenNames(a, b, brand)) throw new Error(`У артикула ${a.articleRaw} разные названия: «${a.name}» и «${b.name}». Уточните соответствие перед объединением.`);
    const row = (warehouseHasSales ? b?.row : a?.row) ?? ++lastRow;
    const value = (source, item, col) => item && col ? (parseNumber(sheetCellValue(source.detection.sheet, item.row, col)) ?? 0) : 0;
    const stockOffice = value(office, a, office.detection.columns.stock);
    const stockWarehouse = value(warehouse, b, warehouse.detection.columns.stock);
    const transitOffice = value(office, a, office.detection.columns.inTransit);
    const transitWarehouse = value(warehouse, b, warehouse.detection.columns.inTransit);
    const facts=preserveFacts ? [[office,a],[warehouse,b]].filter(([,item])=>item).map(([src,item])=>parseNumber(sheetCellValue(src.detection.sheet,item.row,src.detection.columns.orderedFact))).filter(v=>v!=null) : [];
    const comments=preserveFacts ? [[office,a],[warehouse,b]].filter(([,item])=>item).map(([src,item])=>asText(sheetCellValue(src.detection.sheet,item.row,src.detection.columns.comment))).filter(Boolean) : [];
    setTextCell(sheet, row, columns.article, b?.articleRaw || a?.articleRaw || "");
    let mergedName = b?.name || a?.name || "";
    if (a && b && [a.name, b.name].some((name) => /^\s*чз/iu.test(name))) mergedName = `ЧЗ + ${mergedName.replace(/^\s*чз\s*\+?\s*/iu, "")}`;
    setTextCell(sheet, row, columns.name, mergedName);
    calculation.salesColumns.forEach((col, index) => setNumericCell(sheet, row, col, value(office, a, office.calculation.salesColumns[index]) + value(warehouse, b, warehouse.calculation.salesColumns[index])));
    for (const field of ["revenue", "previousQuantity"]) setNumericCell(sheet, row, calculation[field], value(office, a, office.calculation[field]) + value(warehouse, b, warehouse.calculation[field]));
    setNumericCell(sheet, row, columns.stock, stockOffice + stockWarehouse);
    setNumericCell(sheet, row, columns.inTransit, transitOffice + transitWarehouse);
    setNumericCell(sheet, row, columns.orderedFact, facts.length ? Number(facts.reduce((s,v)=>s+v,0).toFixed(2)) : null);
    if(a?.ambiguous || b?.ambiguous)comments.push('Проверить: неоднозначная позиция. Сохранена отдельно, данные складов по ней не объединены.');
    setTextCell(sheet, row, columns.comment, [...new Set(comments)].join('; '));
    for (const [field, amount] of Object.entries({ officeStock: stockOffice, warehouseStock: stockWarehouse, officeTransit: transitOffice, warehouseTransit: transitWarehouse })) setNumericCell(sheet, row, extras[field], amount);
  }
  recalculateSourceTable(detection, deliveryWeeks, brandRule(brand), calculation);
  // Refresh source totals too: their old ranges may omit office-only appended rows.
  const boundsAfterMerge = sheetBounds(sheet);
  const productRows = readSourceRows(detection, boundsAfterMerge.maxRow, boundsAfterMerge.maxColumn, brandRule(brand));
  const additiveColumns = [...calculation.salesColumns, calculation.totalQuantity, calculation.revenue, calculation.previousQuantity, calculation.targetStock, columns.stock, columns.inTransit, columns.recommended, ...Object.values(extras)];
  for (let row = headerRow + 1; row <= boundsAfterMerge.maxRow; row += 1) {
    if (isSourceTotalRow(detection, row, boundsAfterMerge.maxColumn)) sumColumns(sheet, row, productRows.map((item) => item.row), additiveColumns);
  }
  // Excel readers use dimension to bound rows/columns, including the appended location data.
  const dimension = firstElement(sheet.xml, "dimension");
  if (dimension) {
    const bounds = sheetBounds(sheet);
    dimension.setAttribute("ref", `A1:${columnNumberToName(bounds.maxColumn)}${bounds.maxRow}`);
  }
  return workbook;
}

/** Prepare editable office-transfer rows. Supplier minima do not apply here. */
export function buildTyumenWarehousePlan(options) {
  const workbook = mergeTyumenSources(options);
  const detection = detectColumns(workbook, "source");
  const extras = tyumenColumns(detection);
  const bounds = sheetBounds(detection.sheet);
  return readSourceRows(detection, bounds.maxRow, bounds.maxColumn, brandRule(options.brand)).map((item) => {
    const officeStock = parseNumber(sheetCellValue(detection.sheet, item.row, extras.officeStock)) ?? 0;
    const warehouseStock = parseNumber(sheetCellValue(detection.sheet, item.row, extras.warehouseStock)) ?? 0;
    if (officeStock < 0 || warehouseStock < 0 || !Number.isInteger(officeStock) || !Number.isInteger(warehouseStock)) throw new Error(`Проверьте остатки «${item.name}»: для перемещения поштучного товара нужны целые неотрицательные значения.`);
    return { article: item.articleRaw, name: item.name, officeStock, warehouseStock, target: roundHalfUp((officeStock + warehouseStock) * 0.25), quantity: warehouseTransferQuantity(officeStock, warehouseStock) };
  });
}
