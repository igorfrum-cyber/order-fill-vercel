import { setNumericCell, setTextCell, sheetBounds } from "./../../infrastructure/excel/xml.js";
import { ensureNovacutanQuantityColumn, prepareNovacutanQuantityCell } from "./../../infrastructure/excel/blanks.js";
import { applyBudgetWorkbookPricing, budgetPrices } from "./../../infrastructure/excel/pricing.js";

export function writeNorthSummaryWorkbook(summary, totals, quantityByKey = null) {
  const quantityForKey = (key) => {
    if (quantityByKey) return Number(quantityByKey.get(key) || 0);
    return Number(totals.get(key)?.totalQuantity || 0);
  };

  if (summary.kind === "novacutan") {
    ensureNovacutanQuantityColumn(summary.detection);
    let adjustedCount = 0;
    for (const position of summary.positions) {
      const total = quantityForKey(position.key);
      const quantity = total > 0 ? total : null;
      prepareNovacutanQuantityCell(summary.detection, position.row);
      setNumericCell(summary.detection.sheet, position.row, position.quantityCol, quantity);
    }
    return { appended: [], adjustedCount };
  }

  if (summary.kind === "splitVariants") {
    for (const position of summary.positions) {
      const total = quantityForKey(position.key);
      setNumericCell(summary.detection.sheet, position.row, position.quantityCol, total > 0 ? total : null);
    }
    return { appended: [], adjustedCount: 0 };
  }

  const sheet = summary.detection.sheet;
  const columns = summary.detection.columns;
  const written = new Set();
  for (const position of summary.positions) {
    const total = quantityForKey(position.key);
    setNumericCell(sheet, position.row, position.quantityCol, total > 0 ? total : null);
    written.add(position.key);
  }

  const missing = Array.from(totals.values()).filter((item) => {
    if (summary.variant && item.variant !== summary.variant) return false;
    return !written.has(item.key) && quantityForKey(item.key) > 0;
  });
  if (!missing.length) return { appended: [], adjustedCount: 0 };

  const { maxRow } = sheetBounds(sheet);
  for (const [index, item] of missing.entries()) {
    const row = maxRow + index + 1;
    const quantity = Number(quantityForKey(item.key).toFixed(2));
    setTextCell(sheet, row, columns.article, item.articleRaw || item.key);
    setTextCell(sheet, row, columns.name, item.name);
    if (columns.unit) setTextCell(sheet, row, columns.unit, item.unit);
    setNumericCell(sheet, row, columns.quantity, quantity);
  }
  return {
    appended: missing.map((item) => ({
      article: item.articleRaw || item.key,
      name: item.name,
      quantity: Number(quantityForKey(item.key).toFixed(2)),
    })),
    adjustedCount: 0,
  };
}

export function transferItemsForCity(totals, city) {
  return Array.from(totals.values())
    .map((item) => ({
      article: item.articleRaw || "",
      name: item.name,
      unit: "шт",
      quantity: item.cities.get(city.key) || 0,
    }))
    .filter((item) => item.quantity > 0);
}

export function transferFilesFromTotals(totals, cities) {
  return cities
    .filter((city) => city.key !== "tyumen")
    .map((city) => ({
      city,
      fileName: `Заказ на перемещение ${city.label}.xlsx`,
      items: transferItemsForCity(totals, city),
    }));
}

export function budgetNorthWorkbookPricing(summary, entries = []) {
  const positions = new Map(summary.positions.map(p=>[p.key,p]));
  const relevant = entries.filter(e=>(e.group || 'main')===(summary.variant || 'main'));
  const mapped = relevant.map(e=>{
    const p=positions.get(e.key);
    if (!p) throw new Error(`Позиция отсутствует в общем бланке: ${e.name}. Проверьте шаблоны городов.`);
    const price=budgetPrices(summary.detection,p.row,p.quantityCol).find(v=>v.label===e.pricing?.label);
    if (!price && e.quantity>0) throw new Error(`Не найдена выбранная колонка цены в общем бланке: ${e.name}`);
    return {...e,blankRow:p.row,quantityCol:p.quantityCol,pricing:{...e.pricing,column:price?.column}};
  });
  return applyBudgetWorkbookPricing(summary.workbook,summary.detection.sheetName || summary.detection.sheet.name,summary.detection.headerRow,mapped);
}
