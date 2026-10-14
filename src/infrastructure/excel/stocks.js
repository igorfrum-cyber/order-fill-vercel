import { NS_MAIN, cellKey, columnNumberToName, copyCellStyle, elements, firstElement, setNumericCell, setTextCell, sheetBounds, sheetCellValue } from "./xml.js";
import { asText, normalizeHeader } from "./../../shared/text.js";
import { normalizeArticle, normalizeName } from "./../../domain/matching/identity.js";
import { articleNormalizeOptions, brandRule } from "./../../domain/brands/rules.js";

/** Add source stock only when the blank-to-source match is unambiguous. */
export function addOrderBlankStocks(workbook, detection, reportRows, brand) {
  const sheet = workbook.sheets.find(item => item.name === detection.sheetName);
  if (!sheet) throw new Error(`Не найден лист бланка: ${detection.sheetName}`);
  const headerRow = detection.headerRow;
  const rule = brandRule(brand);
  const articles = new Map();
  for (const row of reportRows) {
    const article = normalizeArticle(row.blankArticle, articleNormalizeOptions(rule));
    if (article) articles.set(article, (articles.get(article) || 0) + 1);
  }

  const existing = Array.from({ length: sheetBounds(sheet).maxColumn }, (_, index) => index + 1)
    .find(col => normalizeHeader(sheetCellValue(sheet, headerRow, col)) === 'остаток');
  const angioColumn = brand === 'angiopharm' && ['', 'остаток'].includes(normalizeHeader(sheetCellValue(sheet, headerRow, 10)))
    && reportRows.every(row => {
      const cell = sheet.cells.get(cellKey(row.blankRow, 10));
      return !cell || cell.value === '' || typeof cell.value === 'number';
    }) ? 10 : null;
  const column = angioColumn || existing || sheetBounds(sheet).maxColumn + 1;

  copyCellStyle(sheet, headerRow, detection.columns.quantity, column);
  setTextCell(sheet, headerRow, column, 'Остаток');
  for (const row of reportRows) {
    const article = normalizeArticle(row.blankArticle, articleNormalizeOptions(rule));
    const sourceArticle = normalizeArticle(row.sourceArticle, articleNormalizeOptions(rule));
    const sameItem = article
      ? article === sourceArticle && articles.get(article) === 1
      : Boolean(row.blankName && normalizeName(row.blankName) === normalizeName(row.sourceName));
    const rawStock = asText(row.stock).replace(/\s+/g, '').replace(',', '.');
    const stock = rawStock && /^-?\d+(?:\.\d+)?$/.test(rawStock) ? Number(rawStock) : null;
    const reliable = row.sourceRow != null && !row.duplicate && !row.status?.startsWith('warning')
      && sameItem && stock != null && Number.isFinite(stock);
    setNumericCell(sheet, row.blankRow, column, reliable ? stock : null);
  }

  let cols = firstElement(sheet.xml, 'cols');
  if (!cols) {
    cols = sheet.xml.createElementNS(NS_MAIN, 'cols');
    const sheetData = firstElement(sheet.xml, 'sheetData');
    sheet.xml.documentElement.insertBefore(cols, sheetData);
  }
  {
    let width = elements(cols, 'col').find(node => Number(node.getAttribute('min')) === column && Number(node.getAttribute('max')) === column);
    if (!width) {
      width = sheet.xml.createElementNS(NS_MAIN, 'col');
      width.setAttribute('min', String(column));
      width.setAttribute('max', String(column));
      cols.appendChild(width);
    }
    width.setAttribute('width', '14');
    width.setAttribute('customWidth', '1');
  }
  const dimension = firstElement(sheet.xml, 'dimension');
  if (dimension) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    dimension.setAttribute('ref', `A1:${columnNumberToName(maxColumn)}${maxRow}`);
  }
  return workbook;
}
