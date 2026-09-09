import { strFromU8, strToU8, unzipSync, zipSync } from "fflate";
import { DOMParser, XMLSerializer } from "@xmldom/xmldom";

const ARTICLE_TRANSLATION = new Map([
  ["А", "A"], ["В", "B"], ["Е", "E"], ["К", "K"], ["М", "M"], ["Н", "H"], ["О", "O"],
  ["Р", "P"], ["С", "C"], ["Т", "T"], ["Х", "X"], ["У", "Y"], ["а", "A"], ["в", "B"],
  ["е", "E"], ["к", "K"], ["м", "M"], ["н", "H"], ["о", "O"], ["р", "P"], ["с", "C"],
  ["т", "T"], ["х", "X"], ["у", "Y"],
]);

const MONTHS_RU = ["", "январь", "февраль", "март", "апрель", "май", "июнь", "июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь"];
const NS_MAIN = "http://schemas.openxmlformats.org/spreadsheetml/2006/main";
const XML_PARSER = new DOMParser();
const XML_SERIALIZER = new XMLSerializer();
const BRAND_RULES = {
  angiopharm: {
    label: "ANGIOPHARM",
    adjustment: "box",
    adjustmentLabel: "Шт. в коробке",
    adjustmentComment: "до коробки",
  },
  christina: {
    label: "CHRISTINA",
    adjustment: "multiple",
    multiple: 3,
    adjustmentLabel: "Кратность",
    adjustmentComment: "до кратности 3",
  },
  levissime: {
    label: "LeviSsime",
    adjustment: "box",
    adjustmentLabel: "Кол-во в уп.",
    adjustmentComment: "до коробки",
    articlePrefixAliases: ["MT"],
    blankQuantityHeader: "order",
    blankBoxHeader: "packageQuantity",
  },
  sothys: {
    label: "SOTHYS",
    adjustment: "none",
    adjustmentLabel: "Без округления",
    preserveArticleHyphen: true,
    blankLayout: "splitVariants",
  },
  novacutan: {
    label: "NOVACUTAN",
    adjustment: "none",
    adjustmentLabel: "Мин. заказ",
    blankLayout: "novacutan",
  },
  skin_synergy: {
    label: "Skin Synergy",
    adjustment: "none",
    adjustmentLabel: "Без округления",
    blankQuantityHeader: "exactQuantity",
    requireUnit: false,
  },
  klapp: {
    label: "KLAPP",
    adjustment: "nearestMultiple",
    multiple: 3,
    adjustmentLabel: "Кратность",
    adjustmentComment: "до кратности 3",
    blankQuantityHeader: "order",
  },
};

export function loadXlsx(buffer) {
  const files = unzipSync(new Uint8Array(buffer));
  const workbookXml = parseXml(files["xl/workbook.xml"]);
  const relsXml = parseXml(files["xl/_rels/workbook.xml.rels"]);
  const rels = new Map(elements(relsXml, "Relationship").map((node) => [node.getAttribute("Id"), node.getAttribute("Target")]));
  const sharedStrings = files["xl/sharedStrings.xml"] ? parseSharedStrings(parseXml(files["xl/sharedStrings.xml"])) : [];
  const sheets = elements(workbookXml, "sheet").map((node) => {
    const relId = node.getAttribute("r:id");
    const target = rels.get(relId);
    const path = normalizeWorkbookTarget(target);
    const xml = parseXml(files[path]);
    return {
      name: node.getAttribute("name"),
      path,
      xml,
      cells: readSheetCells(xml, sharedStrings),
    };
  });
  return { files, sheets, sharedStrings };
}

export function saveXlsx(workbook) {
  const files = { ...workbook.files };
  for (const sheet of workbook.sheets) {
    files[sheet.path] = strToU8(XML_SERIALIZER.serializeToString(sheet.xml));
  }
  forceFormulaRecalculation(files);
  return zipSync(files, { level: 6 });
}

function forceFormulaRecalculation(files) {
  const workbookPath = "xl/workbook.xml";
  if (files[workbookPath]) {
    const xml = parseXml(files[workbookPath]);
    let calcPr = firstElement(xml, "calcPr");
    if (!calcPr) {
      calcPr = xml.createElementNS(NS_MAIN, "calcPr");
      xml.documentElement.appendChild(calcPr);
    }
    calcPr.setAttribute("calcMode", "auto");
    calcPr.setAttribute("fullCalcOnLoad", "1");
    calcPr.setAttribute("forceFullCalc", "1");
    files[workbookPath] = strToU8(XML_SERIALIZER.serializeToString(xml));
  }

  delete files["xl/calcChain.xml"];
  removeNodesByAttribute(files, "xl/_rels/workbook.xml.rels", "Relationship", "Type", "calcChain");
  removeNodesByAttribute(files, "[Content_Types].xml", "Override", "PartName", "/xl/calcChain.xml");
}

function removeNodesByAttribute(files, path, tagName, attrName, attrValuePart) {
  if (!files[path]) return;
  const xml = parseXml(files[path]);
  for (const node of elements(xml, tagName)) {
    if ((node.getAttribute(attrName) || "").includes(attrValuePart)) {
      node.parentNode?.removeChild(node);
    }
  }
  files[path] = strToU8(XML_SERIALIZER.serializeToString(xml));
}

function parseXml(bytes) {
  return XML_PARSER.parseFromString(strFromU8(bytes), "application/xml");
}

function normalizeWorkbookTarget(target) {
  const clean = target.replace(/^\/+/, "");
  return clean.startsWith("xl/") ? clean : `xl/${clean}`;
}

function elements(root, tagName) {
  // Spreadsheet XML may use a default namespace or prefixes such as s:sheet.
  return Array.from(root.getElementsByTagNameNS("*", tagName));
}

function firstElement(parent, tagName) {
  return parent.getElementsByTagNameNS("*", tagName)[0] || null;
}

function parseSharedStrings(xml) {
  return elements(xml, "si").map((si) => elements(si, "t").map((t) => t.textContent || "").join(""));
}

function readSheetCells(xml, sharedStrings) {
  const map = new Map();
  for (const cell of elements(xml, "c")) {
    const ref = cell.getAttribute("r");
    if (!ref) continue;
    const { row, col } = parseCellRef(ref);
    map.set(cellKey(row, col), {
      row,
      col,
      ref,
      node: cell,
      value: readCellValue(cell, sharedStrings),
    });
  }
  return map;
}

function readCellValue(cell, sharedStrings) {
  const type = cell.getAttribute("t");
  if (type === "inlineStr") return elements(cell, "t").map((node) => node.textContent || "").join("");
  const valueNode = firstElement(cell, "v");
  if (!valueNode) return "";
  const raw = valueNode.textContent || "";
  if (type === "s") return sharedStrings[Number(raw)] ?? "";
  if (type === "b") return raw === "1";
  if (type === "str") return raw;
  const number = Number(raw);
  return Number.isFinite(number) ? number : raw;
}

function cellKey(row, col) {
  return `${row}:${col}`;
}

function parseCellRef(ref) {
  const match = /^([A-Z]+)(\d+)$/.exec(ref);
  if (!match) throw new Error(`Некорректная ссылка ячейки: ${ref}`);
  return { col: columnNameToNumber(match[1]), row: Number(match[2]) };
}

function columnNameToNumber(name) {
  let result = 0;
  for (const ch of name) result = result * 26 + ch.charCodeAt(0) - 64;
  return result;
}

function columnNumberToName(number) {
  let result = "";
  let current = number;
  while (current > 0) {
    const mod = (current - 1) % 26;
    result = String.fromCharCode(65 + mod) + result;
    current = Math.floor((current - 1) / 26);
  }
  return result;
}

function sheetBounds(sheet) {
  let maxRow = 0;
  let maxColumn = 0;
  for (const cell of sheet.cells.values()) {
    maxRow = Math.max(maxRow, cell.row);
    maxColumn = Math.max(maxColumn, cell.col);
  }
  return { maxRow, maxColumn };
}

function sheetCellValue(sheet, row, col) {
  return sheet.cells.get(cellKey(row, col))?.value ?? "";
}

function refreshCellValue(sheet, row, col) {
  const cell = sheet.cells.get(cellKey(row, col));
  if (cell) cell.value = readCellValue(cell.node, []);
}

export function asText(value) {
  if (value == null) return "";
  return String(value).normalize("NFC").replace(/\n/g, " ").trim();
}

export function normalizeHeader(value) {
  return asText(value).toLowerCase().replaceAll("ё", "е").replace(/[^\p{L}\p{N}%]+/gu, " ").replace(/\s+/g, " ").trim();
}

export function normalizeArticle(value, options = {}) {
  const allowed = options.preserveHyphen ? /[^A-Z0-9-]/g : /[^A-Z0-9]/g;
  return asText(value).replace(/[АВЕКМНОРСТХУавекмнорстху]/g, (ch) => ARTICLE_TRANSLATION.get(ch) || ch).toUpperCase().replace(allowed, "");
}

export function normalizeName(value) {
  return normalizeHeader(value).replace(/\bан\b/g, " ").replace(/\bangiopharm\b/g, " ").replace(/\s+/g, " ").trim();
}

export function parseNumber(value) {
  if (value == null || asText(value) === "") return null;
  const normalized = asText(value).replace(/\s+/g, "").replace(",", ".");
  const number = Number(normalized);
  if (Number.isFinite(number)) return number;
  const match = normalized.match(/-?\d+(?:\.\d+)?/);
  if (!match) return null;
  const extracted = Number(match[0]);
  return Number.isFinite(extracted) ? extracted : null;
}

export function roundHalfUp(value) {
  return Math.floor(value + 0.5);
}

function addMonths(date, months) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + months, 1));
}

function lastDayOfMonth(date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + 1, 0));
}

export function parseOrderMonth(value) {
  const match = /^(\d{4})-(\d{2})$/.exec(value || "");
  if (!match) throw new Error("Выберите месяц заказа.");
  return new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, 1));
}

export function formatDate(date) {
  return `${String(date.getUTCDate()).padStart(2, "0")}.${String(date.getUTCMonth() + 1).padStart(2, "0")}.${date.getUTCFullYear()}`;
}

function sameDate(left, right) {
  return left && right && left.getTime() === right.getTime();
}

export function expectedPeriods(orderMonth) {
  const orderDate = parseOrderMonth(orderMonth);
  const mainStart = addMonths(orderDate, -13);
  const mainEnd = lastDayOfMonth(addMonths(orderDate, -2));
  const previousStart = mainStart;
  const previousEnd = lastDayOfMonth(addMonths(mainStart, 2));
  return {
    main: { start: mainStart, end: mainEnd },
    previous: { start: previousStart, end: previousEnd },
    label: `${MONTHS_RU[orderDate.getUTCMonth() + 1]} ${orderDate.getUTCFullYear()}`,
  };
}

export function formatPeriod(period) {
  return `${formatDate(period.start)} - ${formatDate(period.end)}`;
}

function parsePeriodRange(text) {
  const match = /(\d{1,2})\.(\d{1,2})\.(\d{4}).*?(\d{1,2})\.(\d{1,2})\.(\d{4})/.exec(asText(text));
  if (!match) return null;
  const [, d1, m1, y1, d2, m2, y2] = match.map(Number);
  return { start: new Date(Date.UTC(y1, m1 - 1, d1)), end: new Date(Date.UTC(y2, m2 - 1, d2)) };
}

function rangesEqual(left, right) {
  return sameDate(left?.start, right?.start) && sameDate(left?.end, right?.end);
}

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

function sourceMatchers() {
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

function isUrengoySource(workbook) {
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

function detectDeliveryWeeks(workbook) {
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

function isMonthHeader(value) {
  const normalized = normalizeHeader(value);
  if (!/\b20\d{2}\b/.test(normalized)) return false;
  return MONTHS_RU.slice(1).some((month) => normalized.includes(month));
}

function detectUrengoyColumns(detection) {
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

function detectCalculationColumns(detection, options = {}) {
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

function normalizeCategory(value) {
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

function categoryFromCumulative(percent) {
  if (percent <= 50) return "A+";
  if (percent <= 80) return "A";
  if (percent <= 95) return "B";
  return "C";
}

function maxMonthlySales(sheet, row, salesColumns) {
  let max = 0;
  for (const col of salesColumns) {
    const value = parseNumber(sheetCellValue(sheet, row, col));
    if (value != null) max = Math.max(max, value);
  }
  return max;
}

function calculateUrengoyRecommended(sheet, row, urengoyInfo) {
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

function isCombinedChzName(value) {
  return /^\s*чз\s*\+/iu.test(asText(value));
}

function isChzCloneName(value) {
  return normalizeHeader(value).startsWith("чз ") && !isCombinedChzName(value);
}

function comparableChzName(value) {
  return normalizeHeader(value)
    .replace(/^чз\s+/u, "")
    .replace(/\bчз\b/gu, " ")
    .replace(/\bан\b/gu, " ")
    .replace(/\bangiopharm\b/gu, " ")
    .replace(/\s+/gu, " ")
    .trim();
}

function chzNameSimilarity(left, right) {
  return similarity(comparableChzName(left), comparableChzName(right));
}

function calculateTargetNew(values) {
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

function positiveSale(value) {
  return Number.isFinite(value) && value > 0;
}

function newProductCalculation(values, deliveryWeeks) {
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

function isSourceTotalRow(detection, row, maxColumn) {
  const summaryAreaEnd = Math.min(maxColumn, Math.max(1, detection.columns.name - 1));
  for (let col = 1; col <= summaryAreaEnd; col += 1) {
    const text = normalizeHeader(sheetCellValue(detection.sheet, row, col));
    if (text === "итого" || text === "total") return true;
  }
  return false;
}

function readSourceRows(detection, maxRow, maxColumn, rule) {
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

function sumColumns(sheet, targetRow, sourceRowsList, columns) {
  for (const col of columns.filter(Boolean)) {
    const total = sourceRowsList.reduce((sum, row) => sum + (parseNumber(sheetCellValue(sheet, row, col)) ?? 0), 0);
    setNumericCell(sheet, targetRow, col, Number(total.toFixed(2)));
  }
}

function mergeFactAndComment(sheet, sourceRowsList, columns) {
  const facts = sourceRowsList.map((row) => parseNumber(sheetCellValue(sheet, row, columns.orderedFact))).filter((value) => value != null);
  const comments = sourceRowsList.map((row) => asText(sheetCellValue(sheet, row, columns.comment))).filter(Boolean);
  return {
    fact: facts.length ? Number(facts.reduce((sum, value) => sum + value, 0).toFixed(2)) : null,
    comment: Array.from(new Set(comments)).join("; "),
  };
}

function removeWorksheetRows(sheet, rowNumbers) {
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

function monthlyValues(sheet, row, salesColumns) {
  return salesColumns.map((col) => parseNumber(sheetCellValue(sheet, row, col)));
}

function recalculateSourceTable(detection, deliveryWeeks, rule, calculationColumns = detectCalculationColumns(detection)) {
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

function rebuildSourceWithChz(detection, deliveryWeeks, rule, calculationColumns = detectCalculationColumns(detection)) {
  const { sheet, columns } = detection;
  const { maxRow, maxColumn } = sheetBounds(sheet);
  const rows = readSourceRows(detection, maxRow, maxColumn, rule);
  const rowsByArticle = new Map();
  for (const row of rows) {
    if (!row.article) continue;
    if (!rowsByArticle.has(row.article)) rowsByArticle.set(row.article, []);
    rowsByArticle.get(row.article).push(row);
  }

  const rowsToDelete = [];
  for (const group of rowsByArticle.values()) {
    const normalRows = group.filter((row) => !row.isChz);
    const chzRows = group.filter((row) => row.isChz);
    if (!normalRows.length || !chzRows.length) continue;

    const target = normalRows[0];
    const matchingChzRows = chzRows.filter((row) => chzNameSimilarity(target.name, row.name) >= 0.9);
    if (!matchingChzRows.length) continue;

    const sourceRowsList = [target, ...matchingChzRows].map((row) => row.row);
    const mergedName = `ЧЗ + ${target.name}`;
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
    rowsToDelete.push(...matchingChzRows.map((row) => row.row));
  }

  removeWorksheetRows(sheet, rowsToDelete);
  if (calculationColumns.salesColumns.length) recalculateSourceTable(detection, deliveryWeeks, rule, calculationColumns);
}

function blankMatchers(options = {}) {
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

function combinations(arrays) {
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

export function similarity(left, right) {
  const a = normalizeName(left);
  const b = normalizeName(right);
  if (!a || !b) return 0;
  const rows = Array.from({ length: b.length + 1 }, () => 0);
  for (let i = 1; i <= a.length; i += 1) {
    let previous = 0;
    for (let j = 1; j <= b.length; j += 1) {
      const tmp = rows[j];
      rows[j] = a[i - 1] === b[j - 1] ? previous + 1 : Math.max(rows[j], rows[j - 1]);
      previous = tmp;
    }
  }
  return (2 * rows[b.length]) / (a.length + b.length);
}

function readSource(workbook, orderMonth, rule = brandRule("angiopharm"), sourceFileName = "") {
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

const TYUMEN_COLUMNS = {
  officeStock: "Офис: остаток",
  warehouseStock: "Склад: остаток",
  officeTransit: "Офис: в пути",
  warehouseTransit: "Склад: в пути",
};

function tyumenColumns(detection) {
  const result = {};
  for (const [key, header] of Object.entries(TYUMEN_COLUMNS)) {
    for (let col = 1; col <= sheetBounds(detection.sheet).maxColumn; col += 1) {
      if (normalizeHeader(sheetCellValue(detection.sheet, detection.headerRow, col)) === normalizeHeader(header)) result[key] = col;
    }
  }
  return result;
}

function tyumenProductKey(item, brand) {
  if (item.article) return `article:${item.article}`;
  if (brand === "novacutan") return `name:${novacutanPositionKey(item.name)}`;
  return `name:${normalizeName(item.name.replace(/^\s*чз\s*\+?\s*/iu, ""))}`;
}

function tyumenRows(workbook, brand, options = {}) {
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
    if (byKey.has(key)) throw new Error(`Неоднозначная позиция «${item.name}»: повторяется в одной таблице Тюмени. Проверьте строки ${byKey.get(key).row} и ${item.row}.`);
    byKey.set(key, item);
  }
  return { detection, calculation, byKey };
}

function compatibleTyumenNames(left, right) {
  const clean = (name) => normalizeName(name.replace(/^\s*чз\s*\+?\s*/iu, ""));
  return clean(left) === clean(right);
}

function alignTyumenKeys(office, warehouse) {
  // Exact name fallback is allowed only when an article is missing on one side.
  // Distinct known articles must never be merged just because names coincide.
  for (const [key, item] of [...office.byKey]) {
    if (warehouse.byKey.has(key)) continue;
    const candidates = [...warehouse.byKey].filter(([, other]) => (!item.article || !other.article) && compatibleTyumenNames(item.name, other.name));
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
 * Ambiguous identities and mismatched periods fail instead of guessing.
 */
export function mergeTyumenSources({ officeWorkbook, warehouseWorkbook, brand = "angiopharm", officeFileName = "", warehouseFileName = "" }) {
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
  alignTyumenKeys(office, warehouse);
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
    if (a && b && !compatibleTyumenNames(a.name, b.name)) throw new Error(`У артикула ${a.articleRaw} разные названия: «${a.name}» и «${b.name}». Уточните соответствие перед объединением.`);
    const row = (warehouseHasSales ? b?.row : a?.row) ?? ++lastRow;
    const value = (source, item, col) => item && col ? (parseNumber(sheetCellValue(source.detection.sheet, item.row, col)) ?? 0) : 0;
    const stockOffice = value(office, a, office.detection.columns.stock);
    const stockWarehouse = value(warehouse, b, warehouse.detection.columns.stock);
    const transitOffice = value(office, a, office.detection.columns.inTransit);
    const transitWarehouse = value(warehouse, b, warehouse.detection.columns.inTransit);
    setTextCell(sheet, row, columns.article, b?.articleRaw || a?.articleRaw || "");
    let mergedName = b?.name || a?.name || "";
    if (a && b && [a.name, b.name].some((name) => /^\s*чз/iu.test(name))) mergedName = `ЧЗ + ${mergedName.replace(/^\s*чз\s*\+?\s*/iu, "")}`;
    setTextCell(sheet, row, columns.name, mergedName);
    calculation.salesColumns.forEach((col, index) => setNumericCell(sheet, row, col, value(office, a, office.calculation.salesColumns[index]) + value(warehouse, b, warehouse.calculation.salesColumns[index])));
    for (const field of ["revenue", "previousQuantity"]) setNumericCell(sheet, row, calculation[field], value(office, a, office.calculation[field]) + value(warehouse, b, warehouse.calculation[field]));
    setNumericCell(sheet, row, columns.stock, stockOffice + stockWarehouse);
    setNumericCell(sheet, row, columns.inTransit, transitOffice + transitWarehouse);
    setNumericCell(sheet, row, columns.orderedFact, null);
    setTextCell(sheet, row, columns.comment, "");
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

/** Whole-piece office top-up to 25% of current stock; never a reverse transfer. */
export function warehouseTransferQuantity(officeStock, warehouseStock) {
  const office = Math.max(0, officeStock);
  const warehouse = Math.max(0, warehouseStock);
  return Math.min(Math.floor(warehouse), Math.max(0, roundHalfUp((office + warehouse) * 0.25) - office));
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

/**
 * Projected surplus after protecting Tyumen demand, capped by stock that can
 * physically pass through the warehouse. Office surplus cannot go north.
 * Transit and planned supply are future availability, not goods ready to ship.
 * Null location data preserves the legacy single-location calculation.
 */
export function northTyumenFreeStock(stock, inTransit, plannedOrder, target, warehouseStock = null, warehouseTransit = null) {
  const free = Math.max(0, stock + inTransit + plannedOrder - target);
  return warehouseStock == null ? free : Math.min(free, Math.max(0, warehouseStock + (warehouseTransit || 0) + plannedOrder));
}

function brandRule(brand) {
  return BRAND_RULES[brand] || BRAND_RULES.angiopharm;
}

function articleNormalizeOptions(rule) {
  return { preserveHyphen: Boolean(rule.preserveArticleHyphen) };
}

function blankDetectionOptions(rule) {
  return {
    requireBox: rule.adjustment === "box",
    requireUnit: rule.requireUnit,
    quantityHeader: rule.blankQuantityHeader,
    boxHeader: rule.blankBoxHeader,
  };
}

function articleKeys(article, rule) {
  const keys = new Set([article]);
  for (const prefix of rule.articlePrefixAliases || []) {
    if (article.startsWith(prefix) && /^\d+$/.test(article.slice(prefix.length))) {
      keys.add(article.slice(prefix.length));
    } else if (/^\d+$/.test(article)) {
      keys.add(`${prefix}${article}`);
    }
  }
  return Array.from(keys).filter(Boolean);
}

function uniqueBySourceRow(items) {
  const seen = new Set();
  return items.filter((item) => {
    if (seen.has(item.rowIndex)) return false;
    seen.add(item.rowIndex);
    return true;
  });
}

function buildSourceContext(source, rule) {
  const sourceIndex = new Map();
  const noArticleItems = [];
  for (const item of source.items) {
    if (item.article) {
      for (const key of articleKeys(item.article, rule)) {
        if (!sourceIndex.has(key)) sourceIndex.set(key, []);
        sourceIndex.get(key).push(item);
      }
    } else {
      noArticleItems.push(item);
    }
  }
  return {
    sourceIndex,
    noArticleItems,
    sourceDuplicateGroups: sourceDuplicateGroups(sourceIndex),
    sourceArticleCount: new Set(source.items.map((item) => item.article).filter(Boolean)).size,
  };
}

function sourceDuplicateGroups(sourceIndex) {
  const seenGroups = new Set();
  const groups = [];
  for (const [article, items] of sourceIndex.entries()) {
    const uniqueItems = uniqueBySourceRow(items);
    if (uniqueItems.length < 2) continue;
    const signature = uniqueItems.map((item) => item.rowIndex).sort((left, right) => left - right).join(":");
    if (seenGroups.has(signature)) continue;
    seenGroups.add(signature);
    groups.push({
      article,
      candidates: duplicateCandidatesForReport(uniqueItems),
    });
  }
  return groups;
}

function candidatesForArticle(sourceContext, article, rule) {
  return uniqueBySourceRow(articleKeys(article, rule).flatMap((key) => sourceContext.sourceIndex.get(key) || []));
}

function duplicateCandidatesForReport(candidates) {
  return candidates.map((item) => ({
    sourceRow: item.rowIndex,
    sourceArticle: item.articleRaw,
    sourceName: item.name,
    recommended: item.recommended,
    rounded: item.rounded,
    stock: item.stock,
    inTransit: item.inTransit,
  }));
}

function calculateAdjustedQuantity(recommended, rule, boxSizeValue) {
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

function budgetSourceMetrics(detection, row) {
  const c = detectCalculationColumns(detection, { required: false });
  if (!c) return { demand: 0, category: '', delivery: 0.25 };
  const values = monthlyValues(detection.sheet, row, c.salesColumns);
  const weeks = Math.max(1, detectDeliveryWeeks({ sheets: [detection.sheet] }) || 1);
  const novelty = newProductCalculation(values, weeks);
  return {
    demand: novelty?.monthlyNeed ?? calculateTargetNew(values),
    category: normalizeCategory(sheetCellValue(detection.sheet, row, c.category)).split('/')[0],
    delivery: weeks * 0.25,
  };
}

// Only arithmetic and same-sheet references are accepted; workbook text is never executed.
function budgetCellFormula(sheet, cell) {
  if (!cell) return '';
  const formulaNode = firstElement(cell.node, 'f');
  let formula = formulaNode?.textContent;
  if (!formula && formulaNode?.getAttribute('t') === 'shared') {
    const master = [...sheet.cells.values()].find(c => {
      const f = firstElement(c.node, 'f');
      return f?.getAttribute('si') === formulaNode.getAttribute('si') && f.textContent;
    });
    if (master) formula = firstElement(master.node, 'f').textContent.replace(/(\$?)([A-Z]+)(\$?)(\d+)/g, (_, ca, col, ra, row) => {
      const c = columnNameToNumber(col) + (ca ? 0 : cell.col-master.col);
      const r = Number(row) + (ra ? 0 : cell.row-master.row);
      return `${ca}${columnNumberToName(c)}${ra}${r}`;
    });
  }
  return formula || '';
}

function budgetCellPrice(sheet, ref, seen = new Set()) {
  if (seen.has(ref)) return null;
  const cell = sheet.cells.get(ref);
  if (!cell) return null;
  const formula = budgetCellFormula(sheet, cell);
  if (!formula) return parseNumber(cell.value);
  seen = new Set([...seen, ref]);
  const aggregate = /^(SUM|SUMPRODUCT)\(([^()]*)\)$/i.exec(formula);
  if (aggregate) {
    const ranges = aggregate[2].split(',').map(part => {
      const refs = part.trim().replace(/\$/g, '').split(':');
      if (!refs.every(r => /^[A-Z]+\d+$/.test(r))) return null;
      const a = parseCellRef(refs[0]), b = parseCellRef(refs[1] || refs[0]);
      if (b.row < a.row || b.col < a.col || (b.row-a.row+1)*(b.col-a.col+1) > 100000) return null;
      const values = [];
      for (let r=a.row; r<=b.row; r++) for (let c=a.col; c<=b.col; c++) values.push(budgetCellPrice(sheet,cellKey(r,c),seen) ?? 0);
      return values;
    });
    if (ranges.every(Boolean)) {
      if (aggregate[1].toUpperCase() === 'SUM') return ranges.flat().reduce((s,v)=>s+v,0);
      if (ranges.every(r=>r.length===ranges[0].length)) return ranges[0].reduce((s,_,i)=>s+ranges.reduce((p,r)=>p*r[i],1),0);
    }
    return parseNumber(cell.value);
  }
  const round = /^ROUND\((.*),\s*(\d+)\)$/i.exec(formula);
  const sourceExpression = round ? round[1] : formula;
  let invalid = false;
  const expression = sourceExpression.replace(/\$?([A-Z]+)\$?(\d+)/g, (_, col, row) => {
    const address = parseCellRef(`${col}${row}`);
    const key = cellKey(address.row, address.col);
    const referenced = sheet.cells.get(key);
    const value = !referenced || (referenced.value === '' && !budgetCellFormula(sheet, referenced)) ? 0 : budgetCellPrice(sheet, key, seen);
    if (value == null) invalid = true;
    return `(${value ?? 0})`;
  }).replace(/%/g, '/100');
  if (invalid || /[^\d.\s()+*/-]/.test(expression)) return parseNumber(cell.value);
  const tokens = expression.match(/\d+(?:\.\d*)?|\.\d+|[()+*/-]/g) || [];
  let i = 0;
  function atom() {
    if (tokens[i] === '-') { i++; return -atom(); }
    if (tokens[i] === '+') { i++; return atom(); }
    if (tokens[i] === '(') { i++; const v = sum(); if (tokens[i++] !== ')') throw Error(); return v; }
    const v = Number(tokens[i++]); if (!Number.isFinite(v)) throw Error(); return v;
  }
  function product() { let v = atom(); while (tokens[i] === '*' || tokens[i] === '/') { const op = tokens[i++], b = atom(); v = op === '*' ? v*b : v/b; } return v; }
  function sum() { let v = product(); while (tokens[i] === '+' || tokens[i] === '-') { const op = tokens[i++], b = product(); v = op === '+' ? v+b : v-b; } return v; }
  try {
    const v = sum();
    if (i !== tokens.length || !Number.isFinite(v)) return null;
    return round ? Math.round((v + Number.EPSILON) * 10 ** Number(round[2])) / 10 ** Number(round[2]) : v;
  } catch { return null; }
}

function budgetPrices(detection, row, quantityCol) {
  const result = [];
  if (detection.blocks) {
    const block = detection.blocks.find(b => b.quantity === quantityCol);
    if (!block) return result;
    const col = block.quantity - 1;
    const label = asText(sheetCellValue(detection.sheet,detection.headerRow,col));
    if (!/цена|price/i.test(label)) return result;
    return [{ id:'block-price', column:col, label:`${label} (по блоку объема)`, price:budgetCellPrice(detection.sheet,cellKey(row,col)), quantityCol }];
  }
  const { maxColumn } = sheetBounds(detection.sheet);
  for (let col = 1; col <= maxColumn; col++) {
    const label = asText(sheetCellValue(detection.sheet, detection.headerRow, col));
    if (!/цена|price/i.test(label)) continue;
    const price = budgetCellPrice(detection.sheet, cellKey(row, col));
    result.push({ id: String(col), column:col, label, price, quantityCol });
  }
  return result;
}

export function budgetOrderRules(brand, name, box) {
  const rule = brandRule(brand);
  if (brand === 'novacutan') {
    const unit = novacutanSupplierUnitSize(name);
    const minimum = Number(box) > 0 ? Number(box) : novacutanMinimumQuantity(name);
    const step = unit > 1 ? 1 : 10;
    return { unit, minimum: Math.ceil(minimum / step) * step, step };
  }
  const step = ['multiple', 'nearestMultiple'].includes(rule.adjustment) ? rule.multiple : rule.adjustment === 'box' ? Math.max(1, Number(box) || 1) : 1;
  return { unit: 1, step, minimum: rule.adjustment === 'minimum' ? Math.max(1, Number(box) || 1) : step };
}

export function budgetReportRows(result, brand) {
  return result.reportRows.filter(r => r.editable !== false).map(r => ({
    ...r.budget, ...budgetOrderRules(brand, r.blankName, r.blankBoxSize),
    key: r.key, name: r.blankName, group: result.blankId,
    blankRow: r.blankRow, quantityCol: r.blankQuantityCol,
    inventoryKey: String(r.sourceRow),
    unsafe: Boolean(r.duplicate || r.status.startsWith('warning')),
    stock: parseNumber(r.stock) || 0, transit: parseNumber(r.inTransit) || 0,
    prices: budgetPrices(result.blankDetection, r.blankRow, r.blankQuantityCol),
  }));
}

function setBudgetFormula(sheet, row, col, formula, value) {
  const cell = findOrCreateCell(sheet, row, col);
  while (cell.firstChild) cell.removeChild(cell.firstChild);
  cell.removeAttribute('t');
  const f = sheet.xml.createElementNS(NS_MAIN, 'f');
  f.textContent = formula;
  const v = sheet.xml.createElementNS(NS_MAIN, 'v');
  v.textContent = String(value);
  cell.appendChild(f); cell.appendChild(v);
  sheet.cells.get(cellKey(row,col)).value = value;
}

function budgetInvoicePriceColumn(sheet, headerRow, entry) {
  const qtyRef = `${columnNumberToName(entry.quantityCol)}${entry.blankRow}`;
  const priceColumns = [...sheet.cells.values()].filter(c => c.row === headerRow && /цена|price/i.test(asText(c.value))).map(c=>c.col);
  const referenced = new Set();
  for (const c of sheet.cells.values()) {
    if (c.row !== entry.blankRow) continue;
    const refs = budgetCellFormula(sheet,c).replace(/\$/g,'').match(/[A-Z]+\d+/g) || [];
    if (!refs.includes(qtyRef)) continue;
    for (const ref of refs) {
      const p = parseCellRef(ref);
      if (p.row === entry.blankRow && priceColumns.includes(p.col)) referenced.add(p.col);
    }
  }
  if (referenced.size === 1) return [...referenced][0];
  if (referenced.size > 1) throw new Error(`Несколько цен в формуле суммы: ${entry.name}. Нужно проверить бланк.`);
  const discounted = priceColumns.filter(col => /скид/i.test(asText(sheetCellValue(sheet,headerRow,col))));
  if (discounted.length === 1) return discounted[0];
  if (priceColumns.length === 1) return priceColumns[0];
  // Repeated SOTHYS blocks each use their own price immediately before quantity.
  if (entry.pricing.column === entry.quantityCol-1 && priceColumns.includes(entry.pricing.column)) return entry.pricing.column;
  throw new Error(`Не удалось определить цену для суммы: ${entry.name}. Формулы бланка не изменены.`);
}

/** Apply agreed prices to an export copy only. Repeated downloads never compound discounts. */
export function applyBudgetWorkbookPricing(workbook, sheetName, headerRow, entries = []) {
  if (!entries.length) return workbook;
  if (entries.some(e=>e.quantity>0 && !(e.price>0 && e.pricing?.column))) throw new Error('Не для всех заказанных позиций определена цена. Проверьте выбранную колонку цены.');
  const output = loadXlsx(saveXlsx(workbook));
  const sheet = output.sheets.find(s=>s.name===sheetName);
  if (!sheet) throw new Error('Не найден лист бланка для обновления скидки.');
  const prepared = entries.filter(e=>e.price>0 && e.pricing?.column).map(e=>({ ...e,
    effectiveCol: budgetInvoicePriceColumn(sheet,headerRow,e),
  }));
  const changes = new Map();
  for (const e of prepared) {
    const sourceRef = `${columnNumberToName(e.pricing.column)}${e.blankRow}`;
    let formula = budgetCellFormula(sheet,sheet.cells.get(cellKey(e.blankRow,e.effectiveCol))).replace(/\s|\$/g,'');
    formula = formula.replace(/^ROUND\((.*),2\)$/i,'$1');
    const direct = /^([A-Z]+\d+)\*\(1-([A-Z]+\d+)(\/100)?\)$/i.exec(formula);
    if (!direct || direct[1] !== sourceRef) continue;
    const address = parseCellRef(direct[2]);
    if (address.row >= headerRow) continue;
    const value = e.pricing.discount / (direct[3] ? 1 : 100);
    if (changes.has(direct[2]) && changes.get(direct[2]).value !== value) throw new Error('Разные скидки ссылаются на одну ячейку бланка.');
    changes.set(direct[2],{...address,value});
  }
  for (const change of changes.values()) setNumericCell(sheet,change.row,change.col,change.value);

  const fallback = prepared.filter(e=>Math.abs((budgetCellPrice(sheet,cellKey(e.blankRow,e.effectiveCol)) ?? -1)-e.price)>0.000001);
  if (fallback.length) {
    // Materialize shared formula followers before replacing any group master.
    const shared = [...sheet.cells.values()].filter(c=>firstElement(c.node,'f')?.getAttribute('t')==='shared')
      .map(c=>({c,formula:budgetCellFormula(sheet,c)}));
    for (const {c,formula} of shared) {
      if (!formula) throw new Error('Не удалось прочитать общую формулу цены Excel.');
      const f=firstElement(c.node,'f'); f.textContent=formula;
      for (const attr of ['t','si','ref']) f.removeAttribute(attr);
    }
    const bounds=sheetBounds(sheet);
    const baseCol=bounds.maxColumn+2, discountCol=baseCol+1;
    setTextCell(sheet,headerRow,baseCol,'Цена до нашей скидки');
    setTextCell(sheet,headerRow,discountCol,'Наша скидка, %');
    // Multiple blocks on the same row need independent bases and discount cells.
    const blocks=new Map();
    for (const e of fallback) {
      if (!blocks.has(e.quantityCol)) blocks.set(e.quantityCol,blocks.size);
      const offset=blocks.get(e.quantityCol)*2;
      const b=baseCol+offset,d=discountCol+offset;
      setTextCell(sheet,headerRow,b,'Цена до нашей скидки');
      setTextCell(sheet,headerRow,d,'Наша скидка, %');
      setNumericCell(sheet,e.blankRow,b,e.pricing.basePrice);
      setNumericCell(sheet,e.blankRow,d,e.pricing.discount);
      const formula=`ROUND(${columnNumberToName(b)}${e.blankRow}*(1-${columnNumberToName(d)}${e.blankRow}/100),2)`;
      setBudgetFormula(sheet,e.blankRow,e.effectiveCol,formula,e.price);
      if (e.effectiveCol===e.pricing.column) setTextCell(sheet,headerRow,e.effectiveCol,'Цена с учетом нашей скидки');
    }
    const dimension=firstElement(sheet.xml,'dimension');
    const updatedBounds=sheetBounds(sheet);
    if (dimension) dimension.setAttribute('ref',`A1:${columnNumberToName(updatedBounds.maxColumn)}${updatedBounds.maxRow}`);
  }
  // Refresh numeric caches as well as preserving Excel's automatic recalculation.
  for (const c of sheet.cells.values()) {
    if (!budgetCellFormula(sheet,c)) continue;
    const value=budgetCellPrice(sheet,cellKey(c.row,c.col));
    if (value==null || !Number.isFinite(value)) continue;
    let v=firstElement(c.node,'v');
    if (!v) {v=sheet.xml.createElementNS(NS_MAIN,'v');c.node.appendChild(v);}
    v.textContent=String(value); c.value=value;
  }
  for (const e of prepared) {
    const actual=budgetCellPrice(sheet,cellKey(e.blankRow,e.effectiveCol));
    if (actual==null || Math.abs(actual-e.price)>0.000001) throw new Error(`Цена в бланке не совпала с расчетом: ${e.name}`);
  }
  return output;
}

function orderForItem(item, rule, adjustmentValue) {
  if (!item.hasOrderedFact) return calculateAdjustedQuantity(item.recommended, rule, adjustmentValue);
  return {
    ...calculateAdjustedQuantity(item.orderedFact, rule, adjustmentValue),
    autoComment: "",
    fromOrderedFact: true,
  };
}

function supplierUnitsFromPieces(quantity, unitSize = 1) {
  const number = Number(quantity || 0);
  const size = Number(unitSize || 1);
  if (!Number.isFinite(number) || number <= 0) return null;
  if (!Number.isFinite(size) || size <= 1) return Number(number.toFixed(2));
  return Math.ceil(number / size);
}

function demandPiecesFromSupplierUnits(quantity, unitSize = 1) {
  const number = Number(quantity || 0);
  const size = Number(unitSize || 1);
  if (!Number.isFinite(number) || number <= 0) return 0;
  return Number((number * (Number.isFinite(size) && size > 0 ? size : 1)).toFixed(2));
}

function convertOrderToSupplierUnits(order, unitSize = 1) {
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

function novacutanOrderForItem(item, rowInfo, rule) {
  return convertOrderToSupplierUnits(orderForItem(item, rule, rowInfo.blankBoxSize), rowInfo.supplierUnitSize);
}

function calculateMultipleAdjustedQuantity(rounded, multiple, comment) {
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

function nearestMultipleValue(value, multiple) {
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

function calculateNearestMultipleAdjustedQuantity(rounded, multiple, comment) {
  const inserted = nearestMultipleValue(rounded, multiple);
  if (inserted == null) return { rounded, inserted: null, autoComment: "", boxAdjusted: false };
  return {
    rounded,
    inserted,
    autoComment: inserted !== rounded ? comment : "",
    boxAdjusted: inserted !== rounded,
  };
}

function extractVolumeKeys(...values) {
  const keys = new Set();
  const text = values.map((value) => normalizeHeader(value)).join(" ");
  const pattern = /(\d+(?:[,.]\d+)?)\s*(мл|ml|гр|г|g)\b/giu;
  for (const match of text.matchAll(pattern)) {
    const amount = Number(match[1].replace(",", "."));
    if (!Number.isFinite(amount)) continue;
    const unit = match[2].toLowerCase();
    const normalizedUnit = unit === "ml" || unit === "мл" ? "мл" : "гр";
    keys.add(`${amount}:${normalizedUnit}`);
  }
  return keys;
}

function volumeAwareSimilarity(blankName, sourceName, blankUnit = "") {
  const base = similarity(blankName, sourceName);
  const blankVolumes = extractVolumeKeys(blankName, blankUnit);
  const sourceVolumes = extractVolumeKeys(sourceName);
  if (!blankVolumes.size || !sourceVolumes.size) return base;

  const hasMatch = Array.from(blankVolumes).some((key) => sourceVolumes.has(key));
  if (hasMatch) return Math.min(1, base + 0.06);
  return Math.max(0, base - 0.35);
}

function chooseCandidate(candidates, blankName, blankUnit = "") {
  return candidates.map((item) => ({ item, score: volumeAwareSimilarity(blankName, item.name, blankUnit) })).sort((left, right) => right.score - left.score)[0];
}

function chooseNameFallback(candidates, blankName, blankUnit = "") {
  const scored = candidates.filter((item) => !item.article).map((item) => ({ item, score: volumeAwareSimilarity(blankName, item.name, blankUnit) })).sort((left, right) => right.score - left.score);
  if (!scored.length) return { item: null, score: 0 };
  const bestNonpositive = scored.find((entry) => entry.item.rounded <= 0 && entry.score >= 0.72);
  if (bestNonpositive) return bestNonpositive;
  if (scored.length > 1 && scored[0].score - scored[1].score < 0.08) return { item: null, score: scored[0].score };
  if (scored[0].score < 0.72) return { item: null, score: scored[0].score };
  return scored[0];
}

function chooseSothysNameFallback(candidates, blankName, blankUnit) {
  const unitText = normalizeHeader(blankUnit);
  const scored = candidates
    .filter((item) => !item.article && (!unitText || normalizeHeader(item.name).includes(unitText)))
    .map((item) => ({ item, score: volumeAwareSimilarity(blankName, item.name, blankUnit) }))
    .sort((left, right) => right.score - left.score);
  if (!scored.length || scored[0].score < 0.72) return { item: null, score: scored[0]?.score || 0 };
  if (scored.length > 1 && scored[0].score - scored[1].score < 0.08) return { item: null, score: scored[0].score };
  return scored[0];
}

function genericBlankPositions(blank, blankId, blankLabel, rule) {
  const rows = [];
  const { maxRow } = sheetBounds(blank.sheet);
  for (let row = blank.headerRow + 1; row <= maxRow; row += 1) {
    const blankArticleRaw = asText(sheetCellValue(blank.sheet, row, blank.columns.article));
    const blankArticle = normalizeArticle(blankArticleRaw, articleNormalizeOptions(rule));
    if (!blankArticle) continue;
    rows.push({
      key: `${blankId}:${row}`,
      blankId,
      blankLabel,
      blankRow: row,
      blankQuantityCol: blank.columns.quantity,
      blankArticleRaw,
      blankArticle,
      blankName: asText(sheetCellValue(blank.sheet, row, blank.columns.name)),
      blankUnit: asText(sheetCellValue(blank.sheet, row, blank.columns.unit)),
      blankBoxSize: rule.adjustment === "box" ? sheetCellValue(blank.sheet, row, blank.columns.boxSize) : rule.multiple,
    });
  }
  return rows;
}

function duplicateArticleWarnings(positions) {
  const groups = new Map();
  for (const position of positions) {
    if (!position.blankArticle) continue;
    if (!groups.has(position.blankArticle)) groups.set(position.blankArticle, []);
    groups.get(position.blankArticle).push(position.blankRow);
  }
  return Array.from(groups.entries())
    .filter(([, rows]) => rows.length > 1)
    .map(([article, rows]) => ({ type: "blank_duplicate_article", article, rows }));
}

function sourceItemsForMissingBlank(source) {
  return source.items
    .filter((item) => item.recommended > 0)
    .map((item) => ({
      sourceRow: item.rowIndex,
      sourceArticle: item.articleRaw,
      sourceName: item.name,
      recommended: item.recommended,
      rounded: item.rounded,
      stock: item.stock,
      inTransit: item.inTransit,
      hasOrderedFact: item.hasOrderedFact,
      orderedFact: item.hasOrderedFact ? item.orderedFact : null,
      sourceComment: item.sourceComment,
    }));
}

function detectSothysVariantBlank(workbook) {
  let best = null;
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 220); row += 1) {
      const blocks = [];
      for (let col = 1; col <= maxColumn - 4; col += 1) {
        const articleHeader = normalizeHeader(sheetCellValue(sheet, row, col));
        const volumeHeader = normalizeHeader(sheetCellValue(sheet, row, col + 1));
        const quantityHeader = normalizeHeader(sheetCellValue(sheet, row, col + 4));
        if ((articleHeader.includes("арт") || articleHeader.includes("артикул")) && (volumeHeader.includes("объем") || volumeHeader.includes("объм") || volumeHeader.includes("обьм")) && quantityHeader === "заказ") {
          blocks.push({
            article: col,
            volume: col + 1,
            unit: col + 2,
            quantity: col + 4,
          });
        }
      }
      if (blocks.length > (best?.blocks.length || 0)) {
        best = { sheet, sheetName: sheet.name, headerRow: row, blocks };
      }
      if (blocks.length >= 2) return { sheet, sheetName: sheet.name, headerRow: row, blocks };
    }
  }
  if (best?.blocks.length) return best;
  throw new Error("Не удалось найти блоки SOTHYS: Артикул, Объем, Заказ.");
}

function sothysPositions(blank, blankId, blankLabel, rule) {
  const rows = [];
  const { maxRow } = sheetBounds(blank.sheet);
  let lastEnglishName = "";
  let lastRussianName = "";
  for (let row = blank.headerRow + 1; row <= maxRow; row += 1) {
    const rowHasHeader = blank.blocks.some((block) => normalizeHeader(sheetCellValue(blank.sheet, row, block.article)).includes("арт"));
    if (rowHasHeader) continue;

    const englishName = asText(sheetCellValue(blank.sheet, row, 2));
    const russianName = asText(sheetCellValue(blank.sheet, row, 3));
    const hasAnyArticle = blank.blocks.some((block) => asText(sheetCellValue(blank.sheet, row, block.article)));
    if (hasAnyArticle && (englishName || russianName)) {
      lastEnglishName = englishName || lastEnglishName;
      lastRussianName = russianName || lastRussianName;
    }

    for (const block of blank.blocks) {
      const blankArticleRaw = asText(sheetCellValue(blank.sheet, row, block.article));
      const blankArticle = normalizeArticle(blankArticleRaw, articleNormalizeOptions(rule));
      if (!blankArticle) continue;
      const nameParts = [englishName || lastEnglishName, russianName || lastRussianName].filter(Boolean);
      const blankName = nameParts.join(" / ");
      const blankUnit = [asText(sheetCellValue(blank.sheet, row, block.volume)), asText(sheetCellValue(blank.sheet, row, block.unit))].filter(Boolean).join(" ");
      rows.push({
        key: `${blankId}:${row}:${block.quantity}`,
        blankId,
        blankLabel,
        blankRow: row,
        blankQuantityCol: block.quantity,
        blankArticleRaw,
        blankArticle,
        blankName,
        blankUnit,
        blankBoxSize: "",
      });
    }
  }
  return rows;
}

function novacutanBlankPositions(blank, blankId, blankLabel) {
  const rows = [];
  const { maxRow } = sheetBounds(blank.sheet);
  for (let row = blank.headerRow + 1; row <= maxRow; row += 1) {
    const blankName = asText(sheetCellValue(blank.sheet, row, blank.columns.name));
    const normalized = normalizeHeader(blankName);
    if (!blankName || normalized === "описание" || normalized === "novacutan") continue;
    const minimumFromBlank = blank.columns.minimum ? parseNumber(sheetCellValue(blank.sheet, row, blank.columns.minimum)) : null;
    rows.push({
      key: `${blankId}:${row}`,
      blankId,
      blankLabel,
      blankRow: row,
      blankQuantityCol: blank.columns.quantity,
      blankArticleRaw: "",
      blankArticle: "",
      blankName,
      blankUnit: "шт",
      blankBoxSize: minimumFromBlank ?? novacutanMinimumQuantity(blankName),
      supplierUnitSize: novacutanSupplierUnitSize(blankName),
    });
  }
  return rows;
}

function novacutanMatchKey(value) {
  const text = normalizeHeader(value)
    .replace(/\bbiopro\b/g, "bio pro")
    .replace(/\bbio\s*pro\b/g, "bio pro");
  if (!text || text.includes("термопакет") || text.includes("хладоэлемент")) return "";
  const hasFillerMask = text.includes("filler") || text.includes("филлер") || text.includes("маск") || text.includes("mask");
  if (hasFillerMask && (text.includes("глаз") || text.includes("eye"))) return "mask-eye";
  if (hasFillerMask && (text.includes("лица") || text.includes("face"))) return "mask-face";
  if (text.includes("sbio")) return "sbio";
  if (text.includes("ybio")) return "ybio";
  if (text.includes("bio pro")) return "bio-pro";
  if (text.includes("prima")) return "prima";
  if (text.includes("master")) return "master";
  if (text.includes("bright") || text.includes("брайт")) return "bright";
  if (text.includes("gentle") || text.includes("джентл") || text.includes("джентел")) return "gentle";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("light")) return "dvs-fbio-light";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("medium")) return "dvs-fbio-medium";
  if (text.includes("fbio") && text.includes("dvs") && text.includes("volume")) return "dvs-fbio-volume";
  if (text.includes("fbio") && text.includes("light")) return "fbio-light";
  if (text.includes("fbio") && text.includes("medium")) return "fbio-medium";
  if (text.includes("fbio") && text.includes("volume")) return "fbio-volume";
  if (text.includes("eye")) return "eye";
  return "";
}

function chooseNovacutanCandidate(sourceItems, blankName, blankUnit = "") {
  const key = novacutanMatchKey(blankName);
  if (key) {
    const keyed = sourceItems.filter((item) => novacutanMatchKey(item.name) === key);
    const blankText = normalizeHeader(blankName);
    const beautyBoxAligned = keyed.filter((item) => {
      const sourceText = normalizeHeader(item.name);
      const blankIsBeautyBox = blankText.includes("бьюти") || blankText.includes("beauty");
      const sourceIsBeautyBox = sourceText.includes("бьюти") || sourceText.includes("beauty");
      return blankIsBeautyBox === sourceIsBeautyBox;
    });
    const nonBonus = (beautyBoxAligned.length ? beautyBoxAligned : keyed).filter((item) => !normalizeHeader(item.name).includes("бонус"));
    const pool = nonBonus.length ? nonBonus : beautyBoxAligned.length ? beautyBoxAligned : keyed;
    if (pool.length) {
      return pool
        .map((item) => ({ item, score: Math.max(0.92, volumeAwareSimilarity(blankName, item.name, blankUnit)) }))
        .sort((left, right) => {
          const positiveDiff = Number(right.item.recommended > 0) - Number(left.item.recommended > 0);
          if (positiveDiff) return positiveDiff;
          return right.score - left.score;
        })[0];
    }
  }
  return chooseCandidate(sourceItems, blankName, blankUnit);
}

export function fillWorkbook({ sourceWorkbook, sourceFileName = "", blankWorkbook, orderMonth, brand = "angiopharm", blankId = "blank", blankLabel = "" }) {
  const rule = brandRule(brand);
  const source = readSource(sourceWorkbook, orderMonth, rule, sourceFileName);
  const sourceContext = buildSourceContext(source, rule);
  if (rule.blankLayout === "novacutan") {
    return fillNovacutanWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel });
  }
  if (rule.blankLayout === "splitVariants") {
    return fillSplitVariantWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel });
  }
  const blank = rule.label === "KLAPP"
    ? detectKlappBlank(blankWorkbook)
    : detectColumns(blankWorkbook, "blank", blankDetectionOptions(rule));
  if (!blank) throw new Error("Не удалось распознать бланк KLAPP: не нашел основной лист с колонками Артикул, Наименование и Заказ.");
  const blankPositions = genericBlankPositions(blank, blankId, blankLabel, rule);
  const blankWarnings = duplicateArticleWarnings(blankPositions);
  const reportRows = [];
  let filled = 0;
  let leftBlank = 0;
  let suspicious = 0;
  let unmatched = 0;
  let duplicates = 0;
  for (const rowInfo of blankPositions) {
    const row = rowInfo.blankRow;
    const { blankArticle, blankName, blankUnit, blankBoxSize } = rowInfo;
    let selected;
    let score;
    let status;
    let order;
    const candidates = candidatesForArticle(sourceContext, blankArticle, rule);
    if (!candidates.length) {
      const fallback = chooseNameFallback(sourceContext.noArticleItems, blankName, blankUnit);
      if (!fallback.item) {
        unmatched += 1;
        setNumericCell(blank.sheet, row, blank.columns.quantity, null);
        reportRows.push(makeUnmatchedReportRow(rowInfo, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      selected = fallback.item;
      score = fallback.score;
      if (selected.rounded > 0) {
        suspicious += 1;
        order = orderForItem(selected, rule, blankBoxSize);
        setNumericCell(blank.sheet, row, blank.columns.quantity, null);
        reportRows.push(makeReportRow("warning_name_only", rowInfo, selected, score, { ...order, inserted: null, autoComment: "" }, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      status = "matched_by_name";
    } else {
      const isDuplicate = candidates.length > 1;
      if (isDuplicate) duplicates += 1;
      rowInfo.duplicateCandidates = isDuplicate ? duplicateCandidatesForReport(candidates) : [];
      const candidate = chooseCandidate(candidates, blankName, blankUnit);
      selected = candidate.item;
      score = candidate.score;
      status = "matched";
      if (score < 0.32) {
        status = "warning_name_differs";
        suspicious += 1;
      }
    }
    order = orderForItem(selected, rule, blankBoxSize);
    rowInfo.duplicate = candidates.length > 1;
    if (order.inserted == null) {
      setNumericCell(blank.sheet, row, blank.columns.quantity, null);
      leftBlank += 1;
      status = "left_blank_nonpositive";
    } else {
      setNumericCell(blank.sheet, row, blank.columns.quantity, order.inserted);
      filled += 1;
    }
    reportRows.push(makeReportRow(status, rowInfo, selected, score, order, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
  }
  return {
    blankId,
    blankLabel,
    sourceWorkbook,
    sourceDetection: source.detection,
    blankWorkbook,
    blankDetection: blank,
    sourceItemsForMissingBlank: sourceItemsForMissingBlank(source),
    sourceDuplicateGroups: sourceContext.sourceDuplicateGroups,
    summary: {
      filled,
      leftBlank,
      suspicious,
      unmatched,
      duplicates,
      sourceItems: source.items.length,
      sourceArticles: sourceContext.sourceArticleCount,
      sourceSheet: source.detection.sheetName,
      sourceHeaderRow: source.detection.headerRow,
      blankSheet: blank.sheetName,
      blankHeaderRow: blank.headerRow,
      blankWarnings,
      blankDuplicateArticles: blankWarnings.length,
      brand: rule.label,
      adjustmentLabel: rule.adjustmentLabel,
      ...source.periodInfo,
    },
    reportRows,
  };
}

function fillNovacutanWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel }) {
  const blank = detectNovacutanBlank(blankWorkbook);
  if (!blank) throw new Error("Не удалось распознать бланк NOVACUTAN: не нашел описание товара и колонку «кол-во» или «Заказ от».");
  ensureNovacutanQuantityColumn(blank);
  const blankPositions = novacutanBlankPositions(blank, blankId, blankLabel);
  const reportRows = [];
  let filled = 0;
  let leftBlank = 0;
  let suspicious = 0;
  let unmatched = 0;
  let duplicates = 0;

  for (const rowInfo of blankPositions) {
    const candidate = chooseNovacutanCandidate(source.items, rowInfo.blankName, rowInfo.blankUnit);
    if (!candidate?.item || candidate.score < 0.72) {
      unmatched += 1;
      prepareNovacutanQuantityCell(blank, rowInfo.blankRow);
      setNumericCell(blank.sheet, rowInfo.blankRow, blank.columns.quantity, null);
      reportRows.push(makeUnmatchedReportRow(rowInfo, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
      continue;
    }

    const order = novacutanOrderForItem(candidate.item, rowInfo, rule);
    let status = "matched_by_name";
    if (candidate.score < 0.85) {
      status = "warning_name_differs";
      suspicious += 1;
    }

    if (order.inserted == null) {
      prepareNovacutanQuantityCell(blank, rowInfo.blankRow);
      setNumericCell(blank.sheet, rowInfo.blankRow, blank.columns.quantity, null);
      leftBlank += 1;
      status = "left_blank_nonpositive";
    } else {
      prepareNovacutanQuantityCell(blank, rowInfo.blankRow);
      setNumericCell(blank.sheet, rowInfo.blankRow, blank.columns.quantity, order.inserted);
      filled += 1;
    }

    reportRows.push(makeReportRow(status, rowInfo, candidate.item, candidate.score, order, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
  }

  return {
    blankId,
    blankLabel,
    sourceWorkbook: source.workbook,
    sourceDetection: source.detection,
    blankWorkbook,
    blankDetection: blank,
    sourceItemsForMissingBlank: sourceItemsForMissingBlank(source),
    sourceDuplicateGroups: sourceContext.sourceDuplicateGroups,
    summary: {
      filled,
      leftBlank,
      suspicious,
      unmatched,
      duplicates,
      sourceItems: source.items.length,
      sourceArticles: sourceContext.sourceArticleCount,
      sourceSheet: source.detection.sheetName,
      sourceHeaderRow: source.detection.headerRow,
      blankSheet: blank.sheetName,
      blankHeaderRow: blank.headerRow,
      blankWarnings: [],
      blankDuplicateArticles: 0,
      brand: rule.label,
      adjustmentLabel: rule.adjustmentLabel,
      ...source.periodInfo,
    },
    reportRows,
  };
}

function fillSplitVariantWorkbook({ source, sourceContext, blankWorkbook, rule, blankId, blankLabel }) {
  const blank = detectSothysVariantBlank(blankWorkbook);
  const blankPositions = sothysPositions(blank, blankId, blankLabel, rule);
  const blankWarnings = duplicateArticleWarnings(blankPositions);
  const reportRows = [];
  let filled = 0;
  let leftBlank = 0;
  let suspicious = 0;
  let unmatched = 0;
  let duplicates = 0;

  for (const position of blankPositions) {
    let selected;
    let score;
    let status;
    let order;
    const candidates = candidatesForArticle(sourceContext, position.blankArticle, rule);
    if (!candidates.length) {
      const fallback = chooseSothysNameFallback(sourceContext.noArticleItems, position.blankName, position.blankUnit);
      if (!fallback.item) {
        unmatched += 1;
        setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, null);
        reportRows.push(makeUnmatchedReportRow(position, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      selected = fallback.item;
      score = fallback.score;
      if (selected.rounded > 0) {
        suspicious += 1;
        order = orderForItem(selected, rule, position.blankBoxSize);
        setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, null);
        reportRows.push(makeReportRow("warning_name_only", position, selected, score, { ...order, inserted: null, autoComment: "" }, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
        continue;
      }
      status = "matched_by_name";
    } else {
      const isDuplicate = candidates.length > 1;
      if (isDuplicate) duplicates += 1;
      position.duplicateCandidates = isDuplicate ? duplicateCandidatesForReport(candidates) : [];
      const candidate = chooseCandidate(candidates, position.blankName, position.blankUnit);
      selected = candidate.item;
      score = candidate.score;
      status = "matched";
      if (score < 0.32) {
        status = "warning_name_differs";
        suspicious += 1;
      }
    }

    order = orderForItem(selected, rule, position.blankBoxSize);
    position.duplicate = candidates.length > 1;
    if (order.inserted == null) {
      setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, null);
      leftBlank += 1;
      status = "left_blank_nonpositive";
    } else {
      setNumericCell(blank.sheet, position.blankRow, position.blankQuantityCol, order.inserted);
      filled += 1;
    }
    reportRows.push(makeReportRow(status, position, selected, score, order, { blankId, blankLabel, adjustmentLabel: rule.adjustmentLabel }));
  }

  return {
    blankId,
    blankLabel,
    sourceWorkbook: source.workbook,
    sourceDetection: source.detection,
    blankWorkbook,
    blankDetection: blank,
    sourceItemsForMissingBlank: sourceItemsForMissingBlank(source),
    sourceDuplicateGroups: sourceContext.sourceDuplicateGroups,
    summary: {
      filled,
      leftBlank,
      suspicious,
      unmatched,
      duplicates,
      sourceItems: source.items.length,
      sourceArticles: sourceContext.sourceArticleCount,
      sourceSheet: source.detection.sheetName,
      sourceHeaderRow: source.detection.headerRow,
      blankSheet: blank.sheetName,
      blankHeaderRow: blank.headerRow,
      blankWarnings,
      blankDuplicateArticles: blankWarnings.length,
      brand: rule.label,
      adjustmentLabel: rule.adjustmentLabel,
      ...source.periodInfo,
    },
    reportRows,
  };
}

function makeUnmatchedReportRow(rowInfo, context) {
  return {
    status: "not_in_source",
    blankId: context.blankId,
    blankLabel: context.blankLabel,
    adjustmentLabel: context.adjustmentLabel,
    key: rowInfo.key || `${context.blankId}:${rowInfo.blankRow}`,
    blankRow: rowInfo.blankRow,
    blankQuantityCol: rowInfo.blankQuantityCol,
    blankArticle: rowInfo.blankArticleRaw,
    blankName: rowInfo.blankName,
    blankUnit: rowInfo.blankUnit,
    blankBoxSize: rowInfo.blankBoxSize,
    sourceRow: null,
    sourceArticle: "",
    sourceName: "",
    hasOrderedFact: false,
    orderedFact: null,
    sourceComment: "",
    stock: "",
    inTransit: "",
    recommended: null,
    rounded: null,
    baseRounded: null,
    inserted: null,
    autoComment: "",
    boxAdjusted: false,
    duplicate: Boolean(rowInfo.duplicate),
    duplicateCandidates: rowInfo.duplicateCandidates || [],
    editable: false,
    similarity: 0,
  };
}

function makeReportRow(status, rowInfo, selected, score, order, context) {
  return {
    budget: selected.budget,
    status,
    blankId: context.blankId,
    blankLabel: context.blankLabel,
    adjustmentLabel: context.adjustmentLabel,
    key: rowInfo.key || `${context.blankId}:${rowInfo.blankRow}`,
    blankRow: rowInfo.blankRow,
    blankQuantityCol: rowInfo.blankQuantityCol,
    blankArticle: rowInfo.blankArticleRaw,
    blankName: rowInfo.blankName,
    blankUnit: rowInfo.blankUnit,
    blankBoxSize: rowInfo.blankBoxSize,
    sourceRow: selected.rowIndex,
    sourceArticle: selected.articleRaw,
    sourceName: selected.name,
    hasOrderedFact: selected.hasOrderedFact,
    orderedFact: selected.hasOrderedFact ? selected.orderedFact : null,
    sourceComment: selected.sourceComment,
    stock: selected.stock,
    inTransit: selected.inTransit,
    recommended: selected.recommended,
    rounded: selected.rounded,
    baseRounded: order.rounded,
    inserted: order.inserted,
    autoComment: order.autoComment,
    boxAdjusted: order.boxAdjusted,
    duplicate: Boolean(rowInfo.duplicate),
    duplicateCandidates: rowInfo.duplicateCandidates || [],
    editable: true,
    similarity: Number(score.toFixed(4)),
  };
}

function findOrCreateCell(sheet, rowNumber, colNumber) {
  const key = cellKey(rowNumber, colNumber);
  const existing = sheet.cells.get(key);
  if (existing) return existing.node;
  const sheetData = firstElement(sheet.xml, "sheetData");
  let row = elements(sheetData, "row").find((node) => Number(node.getAttribute("r")) === rowNumber);
  if (!row) {
    row = sheet.xml.createElementNS(NS_MAIN, "row");
    row.setAttribute("r", String(rowNumber));
    sheetData.appendChild(row);
  }
  const ref = `${columnNumberToName(colNumber)}${rowNumber}`;
  const cell = sheet.xml.createElementNS(NS_MAIN, "c");
  cell.setAttribute("r", ref);
  const cells = elements(row, "c");
  const next = cells.find((node) => parseCellRef(node.getAttribute("r")).col > colNumber);
  if (next) row.insertBefore(cell, next);
  else row.appendChild(cell);
  sheet.cells.set(key, { row: rowNumber, col: colNumber, ref, node: cell, value: "" });
  return cell;
}

function clearCellChildren(cell) {
  while (cell.firstChild) cell.removeChild(cell.firstChild);
  cell.removeAttribute("t");
}

function copyCellStyle(sheet, row, fromCol, toCol) {
  const source = sheet.cells.get(cellKey(row, fromCol))?.node;
  if (!source) return;
  const target = findOrCreateCell(sheet, row, toCol);
  const style = source.getAttribute("s");
  if (style != null) target.setAttribute("s", style);
}

function setNumericCell(sheet, row, col, value) {
  const cell = findOrCreateCell(sheet, row, col);
  clearCellChildren(cell);
  if (value != null) {
    const v = sheet.xml.createElementNS(NS_MAIN, "v");
    v.appendChild(sheet.xml.createTextNode(String(value)));
    cell.appendChild(v);
  }
  const key = cellKey(row, col);
  const record = sheet.cells.get(key);
  if (record) record.value = value ?? "";
}

function setTextCell(sheet, row, col, value) {
  const text = asText(value);
  const cell = findOrCreateCell(sheet, row, col);
  clearCellChildren(cell);
  if (text) {
    cell.setAttribute("t", "inlineStr");
    const inline = sheet.xml.createElementNS(NS_MAIN, "is");
    const node = sheet.xml.createElementNS(NS_MAIN, "t");
    node.appendChild(sheet.xml.createTextNode(text));
    inline.appendChild(node);
    cell.appendChild(inline);
  }
  const key = cellKey(row, col);
  const record = sheet.cells.get(key);
  if (record) record.value = text;
}

function normalizedBaselineQuantity(rowInfo, rule = brandRule("angiopharm")) {
  if (!rule.allowSmallPositiveOrder && Number(rowInfo.recommended) < 1.5) return null;
  return Number(rowInfo.rounded) > 0 ? Number(rowInfo.rounded) : null;
}

export function parseEditValue(value) {
  const text = asText(value);
  if (!text) return null;
  const number = Number(text.replace(",", "."));
  if (!Number.isFinite(number) || number < 0 || !Number.isInteger(number)) throw new Error("Количество должно быть целым неотрицательным числом.");
  return number > 0 ? number : null;
}

export function applyEdits(blankWorkbook, edits) {
  const blank = detectColumns(blankWorkbook, "blank");
  for (const edit of edits) {
    const row = Number(edit.blankRow);
    if (!Number.isInteger(row) || row <= blank.headerRow) continue;
    setNumericCell(blank.sheet, row, blank.columns.quantity, parseEditValue(edit.value));
  }
  return blankWorkbook;
}

export function normalizeOrderValue(value) {
  return parseEditValue(value);
}

export function applyFinalEdits({ blankWorkbook, sourceWorkbook, reportRows, edits, brand = "angiopharm" }) {
  const rule = brandRule(brand);
  const blank = rule.blankLayout === "splitVariants"
    ? detectSothysVariantBlank(blankWorkbook)
    : rule.blankLayout === "novacutan"
      ? detectNovacutanBlank(blankWorkbook)
      : rule.label === "KLAPP"
        ? detectKlappBlank(blankWorkbook)
        : detectColumns(blankWorkbook, "blank", blankDetectionOptions(rule));
  const source = detectColumns(sourceWorkbook, "source");
  const editsByRow = new Map(edits.map((edit) => [edit.key || String(Number(edit.blankRow)), edit]));
  const prepared = [];

  for (const rowInfo of reportRows) {
    const edit = editsByRow.get(rowInfo.key) || editsByRow.get(`${rowInfo.blankId}:${rowInfo.blankRow}`) || editsByRow.get(String(Number(rowInfo.blankRow)));
    if (!edit) continue;

    const quantity = parseEditValue(edit.value);
    const comment = asText(edit.comment);
    const initial = rowInfo.inserted == null ? null : Number(rowInfo.inserted);
    const baseline = normalizedBaselineQuantity(rowInfo, rule);
    const requiresComment = quantity !== baseline;
    const stillAutoComment = rowInfo.autoComment && comment.toLowerCase() === rowInfo.autoComment.toLowerCase();
    const autoCommentAllowed = stillAutoComment && quantity === initial;

    if (requiresComment && (!comment || (stillAutoComment && !autoCommentAllowed))) {
      throw new Error("Если значение в колонке «Вставлено» изменено, нужно заполнить комментарий.");
    }

    const blankRow = Number(rowInfo.blankRow);
    const blankQuantityCol = Number(rowInfo.blankQuantityCol || blank.columns?.quantity);
    const sourceRow = Number(rowInfo.sourceRow);
    const shouldRecordSourceFact = quantity !== normalizedBaselineQuantity(rowInfo, rule);
    const sourceFactValue = shouldRecordSourceFact ? (quantity ?? 0) : null;
    prepared.push({ blankRow, blankQuantityCol, sourceRow, quantity, comment, shouldRecordSourceFact, sourceFactValue });
  }

  for (const edit of prepared) {
    if (Number.isInteger(edit.blankRow) && edit.blankRow > blank.headerRow && Number.isInteger(edit.blankQuantityCol)) {
      setNumericCell(blank.sheet, edit.blankRow, edit.blankQuantityCol, edit.quantity);
    }
    if (Number.isInteger(edit.sourceRow) && edit.sourceRow > source.headerRow) {
      setNumericCell(source.sheet, edit.sourceRow, source.columns.orderedFact, edit.sourceFactValue);
      setTextCell(source.sheet, edit.sourceRow, source.columns.comment, edit.shouldRecordSourceFact ? edit.comment : "");
    }
  }

  return { blankWorkbook, sourceWorkbook };
}

export function outputFileName(originalName, cityName = "") {
  const text = asText(originalName);
  const extension = outputExtension(text);
  const stem = text.replace(/\.(xlsx|xlsm|xls)$/i, "").replace(/[^\p{L}\p{N}_ .-]+/gu, "").trim() || "blank";
  const city = asText(cityName).replace(/[^\p{L}\p{N}_ .-]+/gu, "").trim();
  const cityAlreadyInName = city && normalizeHeader(stem).includes(normalizeHeader(city));
  const namedStem = city && !cityAlreadyInName ? `${stem} ${city}` : stem;
  return `${namedStem} заполненный.${extension}`;
}

export function sourceOutputFileName(originalName) {
  const text = asText(originalName);
  const extension = outputExtension(text);
  const stem = text.replace(/\.(xlsx|xlsm|xls)$/i, "").replace(/[^\p{L}\p{N}_ .-]+/gu, "").trim() || "order";
  return `${stem} заполненная таблица.${extension}`;
}

function outputExtension(fileName) {
  return /\.xlsm$/i.test(asText(fileName)) ? "xlsm" : "xlsx";
}

const NORTH_CITIES = [
  { key: "nizhnevartovsk", label: "Нижневартовск", warehouse: "Склад Нижневартовск", aliases: ["нижневартовск", "вартовск"] },
  { key: "urengoy", label: "Уренгой", warehouse: "Склад Уренгой", aliases: ["новый уренгой", "уренгой"] },
  { key: "surgut", label: "Сургут", warehouse: "Склад Сургут", aliases: ["сургут"] },
  { key: "tyumen", label: "Тюмень", warehouse: "Склад Тюмень", aliases: ["тюмень", "склад доставка"] },
];
const NORTH_TRANSFER_DISPLAY_ORDER = ["surgut", "nizhnevartovsk", "urengoy"];

function northCityFromText(value) {
  const text = normalizeHeader(value);
  if (!text) return null;
  return NORTH_CITIES.find((city) => city.aliases.some((alias) => text.includes(alias))) || null;
}

function detectSourceCity(workbook, fileName = "") {
  for (const sheet of workbook.sheets) {
    const city = northCityFromText(sheet.name);
    if (city) return city;
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 60); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const cityInCell = northCityFromText(sheetCellValue(sheet, row, col));
        if (cityInCell) return cityInCell;
      }
    }
  }
  return northCityFromText(fileName);
}

export function validateNorthTyumenSourceWorkbook(workbook, fileName = "") {
  const city = detectSourceCity(workbook, fileName);
  if (!city) {
    throw new Error("Не удалось понять город в заполненной таблице Тюмени. Загрузите именно тюменскую таблицу или добавьте «Тюмень» в название файла.");
  }
  if (city.key !== "tyumen") {
    throw new Error(`В поле «Заполненная таблица Тюмени» загружена таблица города ${city.label}. Нужна именно Тюмень.`);
  }
  return city;
}

function detectNorthCity(workbook, fileName) {
  for (const sheet of workbook.sheets) {
    const city = northCityFromText(sheet.name);
    if (city) return city;
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 50); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const cityInCell = northCityFromText(sheetCellValue(sheet, row, col));
        if (cityInCell) return cityInCell;
      }
    }
  }
  const cityInName = northCityFromText(fileName);
  if (cityInName) return cityInName;
  throw new Error(`Не понял город в файле «${fileName}». В параметрах или названии файла должен быть Сургут, Нижневартовск/Вартовск, Уренгой или Тюмень.`);
}

function northBlankDetection(workbook, fileName = "", brand = "") {
  if (brand === "novacutan") {
    const novacutan = detectNovacutanBlank(workbook);
    if (!novacutan) throw new Error("Не удалось распознать бланк NOVACUTAN: не нашел описание товара и колонку количества.");
    return { kind: "novacutan", detection: novacutan };
  }

  if (brand === "klapp") {
    const klapp = detectKlappBlank(workbook);
    if (!klapp) throw new Error("Не удалось распознать бланк KLAPP: не нашел основной лист с колонками Артикул, Наименование и Заказ.");
    return { kind: "klapp", detection: klapp };
  }

  if (brand === "skin_synergy") {
    return { kind: "generic", detection: detectNameOnlyNorthBlank(workbook, { quantityHeader: "skinSynergy" }) };
  }

  if (brand === "sothys") {
    return { kind: "splitVariants", detection: detectSothysVariantBlank(workbook) };
  }

  const novacutan = detectNovacutanBlank(workbook);
  if (novacutan) return { kind: "novacutan", detection: novacutan };

  const klapp = detectKlappBlank(workbook);
  if (klapp) return { kind: "klapp", detection: klapp };

  const skinSynergy = detectSkinSynergyNorthBlank(workbook, fileName);
  if (skinSynergy) return { kind: "generic", detection: skinSynergy };

  try {
    return { kind: "generic", detection: detectColumns(workbook, "blank", { requireBox: false, quantityHeader: "anyOrder" }) };
  } catch (error) {
    try {
      return { kind: "generic", detection: detectNameOnlyNorthBlank(workbook) };
    } catch (nameOnlyError) {
      if (brand) throw error;
      return { kind: "splitVariants", detection: detectSothysVariantBlank(workbook) };
    }
  }
}

function detectSkinSynergyNorthBlank(workbook, fileName = "", options = {}) {
  const signal = normalizeHeader(fileName);
  let hasSignal = Boolean(options.force) || signal.includes("skin synergy") || signal.includes("скин синерджи");
  for (const sheet of workbook.sheets) {
    if (hasSignal) break;
    if (normalizeHeader(sheet.name).includes("skin synergy") || normalizeHeader(sheet.name).includes("скин синерджи")) {
      hasSignal = true;
      break;
    }
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 60); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        const text = normalizeHeader(sheetCellValue(sheet, row, col));
        if (text.includes("skin synergy") || text.includes("скин синерджи")) {
          hasSignal = true;
          break;
        }
      }
      if (hasSignal) break;
    }
  }
  if (!hasSignal) return null;

  try {
    return detectNameOnlyNorthBlank(workbook, { quantityHeader: "exactQuantity" });
  } catch {
    return detectNameOnlyNorthBlank(workbook);
  }
}

function detectNameOnlyNorthBlank(workbook, options = {}) {
  const quantityHeader = options.quantityHeader === "skinSynergy" ? "exactQuantity" : options.quantityHeader || "anyOrder";
  try {
    return detectColumns(workbook, "blank", {
      requireArticle: false,
      requireBox: false,
      requireUnit: false,
      quantityHeader,
    });
  } catch (error) {
    if (quantityHeader !== "exactQuantity") throw error;
    return detectColumns(workbook, "blank", {
      requireArticle: false,
      requireBox: false,
      requireUnit: false,
      quantityHeader: "anyOrder",
    });
  }
}

function detectKlappBlank(workbook) {
  const hasKlappSignal = workbook.sheets.some((sheet) => {
    if (normalizeHeader(sheet.name).includes("klapp")) return true;
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 40); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        if (normalizeHeader(sheetCellValue(sheet, row, col)).includes("klapp")) return true;
      }
    }
    return false;
  });
  if (!hasKlappSignal) return null;

  const preferredSheets = [
    ...workbook.sheets.filter((sheet) => normalizeHeader(sheet.name).includes("бланк") && !normalizeHeader(sheet.name).includes("промо")),
    ...workbook.sheets.filter((sheet) => !normalizeHeader(sheet.name).includes("промо")),
  ];
  const seen = new Set();
  for (const sheet of preferredSheets) {
    if (seen.has(sheet.path)) continue;
    seen.add(sheet.path);
    try {
      return detectColumns({ sheets: [sheet] }, "blank", { requireBox: true, quantityHeader: "order" });
    } catch {
      try {
        return detectColumns({ sheets: [sheet] }, "blank", { requireBox: false, quantityHeader: "order" });
      } catch {
        // Проверяем следующий лист.
      }
    }
  }
  return null;
}

function detectNovacutanBlank(workbook) {
  let hasNovacutanSignal = false;
  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    if (normalizeHeader(sheet.name).includes("novacutan")) hasNovacutanSignal = true;
    for (let row = 1; row <= Math.min(maxRow, 80); row += 1) {
      for (let col = 1; col <= maxColumn; col += 1) {
        if (normalizeHeader(sheetCellValue(sheet, row, col)).includes("novacutan")) {
          hasNovacutanSignal = true;
        }
      }
    }
  }
  if (!hasNovacutanSignal) return null;

  for (const sheet of workbook.sheets) {
    const { maxRow, maxColumn } = sheetBounds(sheet);
    for (let row = 1; row <= Math.min(maxRow, 120); row += 1) {
      let name = null;
      let price = null;
      let minimum = null;
      let quantity = null;
      for (let col = 1; col <= maxColumn; col += 1) {
        const header = normalizeHeader(sheetCellValue(sheet, row, col));
        if (header.includes("описание") || header.includes("товар") || header.includes("название")) name = col;
        else if (header.includes("цена")) price = col;
        else if (header.includes("заказ") && header.includes("от")) minimum = col;
        else if (header === "заказ" || header.includes("кол во") || header.includes("количество") || header.includes("qty")) quantity = col;
      }
      if (name && (quantity || minimum)) {
        return {
          sheet,
          sheetName: sheet.name,
          headerRow: row,
          columns: {
            name,
            price,
            minimum,
            quantity: quantity || Math.max(name, minimum || name) + 1,
          },
          quantityColumnCreated: !quantity,
        };
      }
    }
  }

  return null;
}

function ensureNovacutanQuantityColumn(blank) {
  if (!blank.quantityColumnCreated) return;
  copyCellStyle(blank.sheet, blank.headerRow, blank.columns.minimum || blank.columns.name, blank.columns.quantity);
  setTextCell(blank.sheet, blank.headerRow, blank.columns.quantity, "Заказ");
}

function prepareNovacutanQuantityCell(blank, row) {
  if (!blank.quantityColumnCreated) return;
  copyCellStyle(blank.sheet, row, blank.columns.minimum || blank.columns.name, blank.columns.quantity);
}

function northGenericPositions(detection, blankId, blankLabel) {
  const rows = [];
  const { maxRow } = sheetBounds(detection.sheet);
  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    const unit = detection.columns.unit ? asText(sheetCellValue(detection.sheet, row, detection.columns.unit)) : "";
    const articleRaw = detection.columns.article ? asText(sheetCellValue(detection.sheet, row, detection.columns.article)) : "";
    const article = detection.columns.article ? normalizeArticle(articleRaw, { preserveHyphen: true }) : "";
    const key = article || northNameKey(name, unit);
    if (!key) continue;
    rows.push({
      key,
      blankId,
      blankLabel,
      row,
      quantityCol: detection.columns.quantity,
      articleCol: detection.columns.article,
      nameCol: detection.columns.name,
      unitCol: detection.columns.unit,
      articleRaw,
      name,
      unit,
      quantity: parseNumber(sheetCellValue(detection.sheet, row, detection.columns.quantity)) ?? 0,
    });
  }
  return rows;
}

function northNovacutanPositions(detection, blankId, blankLabel) {
  const rows = [];
  const { maxRow } = sheetBounds(detection.sheet);
  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    const minimum = detection.columns.minimum ? parseNumber(sheetCellValue(detection.sheet, row, detection.columns.minimum)) : null;
    const supplierUnitSize = novacutanSupplierUnitSize(name);
    const quantity = demandPiecesFromSupplierUnits(parseNumber(sheetCellValue(detection.sheet, row, detection.columns.quantity)) ?? 0, supplierUnitSize);
    if (!name) continue;
    rows.push({
      key: novacutanPositionKey(name),
      blankId,
      blankLabel,
      row,
      quantityCol: detection.columns.quantity,
      articleCol: null,
      nameCol: detection.columns.name,
      unitCol: null,
      articleRaw: "",
      name,
      unit: "шт",
      quantity,
      novacutanMinimum: minimum ?? novacutanMinimumQuantity(name),
      supplierUnitSize,
    });
  }
  return rows;
}

function novacutanPositionKey(name) {
  const matched = novacutanMatchKey(name);
  return `novacutan:${matched || normalizeName(name).replace(/\bновакутан\b/g, "novacutan")}`;
}

function novacutanMinimumQuantity(name) {
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

function novacutanSupplierUnitSize(name) {
  const key = novacutanMatchKey(name);
  return key === "mask-eye" || key === "mask-face" ? 5 : 1;
}

function novacutanSummaryQuantity(position, total) {
  if (total <= 0) return null;
  const minimum = Number(position.novacutanMinimum || 100);
  return total < minimum ? minimum : Number(total.toFixed(2));
}

function roundSupplierPack10(value) {
  if (value <= 0) return null;
  return Math.round(value / 10) * 10;
}

function novacutanSupplierOrderQuantity(position, supplierNeed) {
  if (supplierNeed <= 0) return null;
  const minimum = Number(position.novacutanMinimum || 100);
  const base = Math.max(Number(supplierNeed), minimum);
  if (Number(position.supplierUnitSize || 1) > 1) return Number(base.toFixed(2));
  return roundSupplierPack10(base);
}

function northSplitVariantPositions(detection, blankId, blankLabel) {
  const rows = [];
  const rule = { preserveArticleHyphen: true, adjustment: "none" };
  for (const position of sothysPositions(detection, blankId, blankLabel, rule)) {
    const quantity = parseNumber(sheetCellValue(detection.sheet, position.blankRow, position.blankQuantityCol)) ?? 0;
    rows.push({
      key: position.blankArticle,
      blankId,
      blankLabel,
      row: position.blankRow,
      quantityCol: position.blankQuantityCol,
      articleCol: null,
      nameCol: null,
      unitCol: null,
      articleRaw: position.blankArticleRaw,
      name: position.blankName,
      unit: position.blankUnit,
      quantity,
    });
  }
  return rows;
}

function northPositions(workbook, blankId, blankLabel, fileName = "", brand = "") {
  const detected = northBlankDetection(workbook, fileName, brand);
  const positions = detected.kind === "novacutan"
    ? northNovacutanPositions(detected.detection, blankId, blankLabel)
    : detected.kind === "splitVariants"
      ? northSplitVariantPositions(detected.detection, blankId, blankLabel)
      : northGenericPositions(detected.detection, blankId, blankLabel);
  const boxMatcher = blankMatchers({ boxHeader: brandRule(brand).blankBoxHeader }).boxSize;
  const boxCol = Array.from({length:sheetBounds(detected.detection.sheet).maxColumn}, (_,i)=>i+1)
    .find(col=>boxMatcher(normalizeHeader(sheetCellValue(detected.detection.sheet,detected.detection.headerRow,col))));
  for (const p of positions) {
    p.budgetPrices = budgetPrices(detected.detection, p.row, p.quantityCol);
    p.blankBoxSize = boxCol ? parseNumber(sheetCellValue(detected.detection.sheet,p.row,boxCol)) : null;
  }
  return { ...detected, positions };
}

function addNorthTotal(totals, item, city) {
  if (!item.key || item.quantity <= 0) return;
  if (!totals.has(item.key)) {
    totals.set(item.key, {
      key: item.key,
      articleRaw: item.articleRaw,
      name: item.name,
      unit: item.unit,
      baseKey: item.baseKey || item.key,
      variant: item.variant || "",
      variantLabel: item.variantLabel || "",
      totalQuantity: 0,
      cities: new Map(),
    });
  }
  const current = totals.get(item.key);
  current.totalQuantity += item.quantity;
  if (!current.name && item.name) current.name = item.name;
  if (!current.unit && item.unit) current.unit = item.unit;
  if (!current.articleRaw && item.articleRaw) current.articleRaw = item.articleRaw;
  current.cities.set(city.key, (current.cities.get(city.key) || 0) + item.quantity);
}

function scopedNorthKey(key, variant = "") {
  return variant ? `${variant}:${key}` : key;
}

function scopedNorthPositions(positions, variant = "", variantLabel = "") {
  if (!variant) return positions;
  return positions.map((position) => ({
    ...position,
    baseKey: position.key,
    key: scopedNorthKey(position.key, variant),
    variant,
    variantLabel,
  }));
}

function writeNorthSummaryWorkbook(summary, totals, quantityByKey = null) {
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

function transferItemsForCity(totals, city) {
  return Array.from(totals.values())
    .map((item) => ({
      article: item.articleRaw || "",
      name: item.name,
      unit: "шт",
      quantity: item.cities.get(city.key) || 0,
    }))
    .filter((item) => item.quantity > 0);
}

function transferFilesFromTotals(totals, cities) {
  return cities
    .filter((city) => city.key !== "tyumen")
    .map((city) => ({
      city,
      fileName: `Заказ на перемещение ${city.label}.xlsx`,
      items: transferItemsForCity(totals, city),
    }));
}

function formatNorthQuantity(value) {
  const number = Number(value || 0);
  if (Number.isInteger(number)) return String(number);
  return String(Number(number.toFixed(2)));
}

function formatNorthCommentQuantity(value) {
  const number = Math.round(Number(value || 0));
  return Number.isFinite(number) && number > 0 ? String(number) : "";
}

function supplierPartsForActual(row, actualValue) {
  const actual = demandPiecesFromSupplierUnits(actualValue, row.supplierUnitSize);
  const northParts = (row.supplierParts || []).filter((part) => part.key !== "tyumen");
  const northNeed = northParts.reduce((sum, part) => sum + Number(part.quantity || 0), 0);
  const tyumenQuantity = Number(Math.max(0, actual - northNeed).toFixed(2));
  return [
    ...(tyumenQuantity > 0 ? [{ key: "tyumen", label: "Тюмень", quantity: tyumenQuantity }] : []),
    ...northParts,
  ];
}

function northTransferParts(row) {
  const quantities = new Map((row.cities || []).map((city) => [city.key, Number(city.quantity || 0)]));
  return NORTH_TRANSFER_DISPLAY_ORDER
    .map((cityKey) => {
      const city = NORTH_CITIES.find((item) => item.key === cityKey);
      return city ? { key: city.key, label: city.label, quantity: quantities.get(city.key) || 0 } : null;
    })
    .filter(Boolean);
}

function northSupplierOrderText(row, actual) {
  const actualRounded = Math.round(Number(actual || 0));
  const neededRounded = Math.round(Number(row.supplierNeed || 0));
  const extraRounded = Math.max(0, actualRounded - neededRounded);
  const unitNote = Number(row.supplierUnitSize || 1) > 1 ? ` коробок по ${Number(row.supplierUnitSize)}` : "";
  if (extraRounded > 0) {
    return `${neededRounded} + ${extraRounded} (${row.budgetComment ? 'сверх потребности' : 'до минимального'}) = ${actualRounded}${unitNote}`;
  }
  return `${formatNorthQuantity(actual)}${unitNote}`;
}

function northPlanCommentForOrderTable(row, actualValue) {
  const lines = [];
  const actual = Number(actualValue || 0);
  const supplierParts = supplierPartsForActual(row, actualValue);
  const tyumenSupplier = supplierParts.find((part) => part.key === "tyumen");
  const transferParts = northTransferParts(row);

  if (actual > 0) lines.push(`Заказать у поставщика: ${northSupplierOrderText(row, actual)}`);
  for (const part of transferParts) {
    if (Number(part.quantity || 0) > 0) lines.push(`Отправить в ${part.label}: ${formatNorthQuantity(part.quantity)}`);
  }
  if (Number(tyumenSupplier?.quantity || 0) > 0) {
    lines.push(`Оставить в Тюмени: ${formatNorthCommentQuantity(tyumenSupplier.quantity)}`);
  }
  if (!lines.length && Number(row.northNeed || 0) > 0) lines.push("Закрывается остатком Тюмени");
  if (row.budgetComment) lines.push(row.budgetComment);
  return lines.join("\n");
}

function novacutanNorthOrderTable(summary, planRows, actualByKey) {
  if (summary.kind !== "novacutan" && !planRows.some(r => r.budgetComment)) return null;
  const rows = [];

  for (const row of planRows) {
    const orderedQuantity = Number(actualByKey.get(row.key) || 0);
    if (orderedQuantity == null) continue;
    if (orderedQuantity <= 0) continue;

    rows.push({
      name: row.name,
      quantity: orderedQuantity,
      comment: northPlanCommentForOrderTable(row, orderedQuantity),
    });
  }

  return {
    fileName: `${summary.kind === 'novacutan' ? 'NOVACUTAN' : 'Север'} заполненная таблица.xlsx`,
    rows,
  };
}

function northNameKey(name, unit = "") {
  const normalized = normalizeName(`${name} ${unit || ""}`);
  return normalized ? `name:${normalized}` : "";
}

function sourceKeyCandidatesForNorth(item, kind) {
  const keys = new Set();
  if (kind === "novacutan") keys.add(novacutanPositionKey(item.name));
  if (!item.articleRaw) keys.add(northNameKey(item.name));
  const preserved = normalizeArticle(item.articleRaw, { preserveHyphen: true });
  const regular = normalizeArticle(item.articleRaw, { preserveHyphen: false });
  if (preserved) keys.add(preserved);
  if (regular) keys.add(regular);
  return Array.from(keys).filter(Boolean);
}

function readNorthTyumenAvailability(workbook, kind) {
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

function allocateNorthNeedByCity(total, tyumenFree) {
  const supplierParts = [];
  const tyumenParts = [];
  let freeLeft = Math.max(0, tyumenFree || 0);

  for (const city of NORTH_CITIES.filter((item) => item.key !== "tyumen")) {
    const quantity = Number(total.cities.get(city.key) || 0);
    if (quantity <= 0) continue;
    const fromTyumen = Math.min(quantity, freeLeft);
    const fromSupplier = quantity - fromTyumen;
    freeLeft -= fromTyumen;
    if (fromTyumen > 0) tyumenParts.push({ key: city.key, label: city.label, quantity: Number(fromTyumen.toFixed(2)) });
    if (fromSupplier > 0) supplierParts.push({ key: city.key, label: city.label, quantity: Number(fromSupplier.toFixed(2)) });
  }

  return {
    fromTyumen: tyumenParts.reduce((sum, part) => sum + part.quantity, 0),
    supplierNorthNeed: supplierParts.reduce((sum, part) => sum + part.quantity, 0),
    supplierParts,
    tyumenParts,
  };
}

function northAvailabilityForTotal(availability, total, kind) {
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

function novacutanNameTokens(value) {
  return normalizeHeader(value)
    .split(/\s+/u)
    .map((token) => token.trim())
    .filter((token) => token && token !== "novacutan" && token !== "мл" && token !== "гр");
}

function novacutanNameScore(left, right, unit = "") {
  const base = volumeAwareSimilarity(left, right, unit);
  const leftTokens = new Set(novacutanNameTokens(left));
  const rightTokens = new Set(novacutanNameTokens(right));
  if (!leftTokens.size || !rightTokens.size) return base;
  const intersection = Array.from(leftTokens).filter((token) => rightTokens.has(token)).length;
  const coverage = intersection / Math.min(leftTokens.size, rightTokens.size);
  if (coverage >= 0.75) return Math.max(base, 0.9);
  return base;
}

function defaultNorthActualSupplierOrder(summary, position, supplierNeed) {
  if (supplierNeed <= 0) return null;
  if (summary.kind === "novacutan") return novacutanSupplierOrderQuantity(position, supplierNeed);
  if (summary.kind === "klapp") return nearestMultipleValue(supplierNeed, 3);
  return Number(supplierNeed.toFixed(2));
}

function northTotalCityParts(total) {
  return NORTH_CITIES
    .map((city) => ({
      key: city.key,
      label: city.label,
      quantity: Number((total.cities.get(city.key) || 0).toFixed(2)),
    }))
    .filter((city) => city.quantity > 0);
}

function northPlanRowFromTotal(summary, total, position, tyumen = null, tyumenFallbackOrder = 0) {
  const tyumenUploadedOrder = Number(total.cities.get("tyumen") || 0);
  const tyumenPlannedOrder = Number(total.cities.has("tyumen") ? tyumenUploadedOrder : tyumenFallbackOrder);
  const tyumenSupplierNeed = Math.max(0, tyumenPlannedOrder);
  const tyumenStock = Number(tyumen?.stock || 0);
  const tyumenInTransit = Number(tyumen?.inTransit || 0);
  const tyumenTarget = Number(tyumen?.targetStock || 0);
  const tyumenFree = northTyumenFreeStock(tyumenStock, tyumenInTransit, tyumenPlannedOrder, tyumenTarget, tyumen?.warehouseStock, tyumen?.warehouseTransit);
  const allocation = allocateNorthNeedByCity(total, tyumenFree);
  const supplierUnitSize = Number(position.supplierUnitSize || 1);
  const supplierParts = [];
  if (tyumenSupplierNeed > 0) supplierParts.push({ key: "tyumen", label: "Тюмень", quantity: Number(tyumenSupplierNeed.toFixed(2)) });
  supplierParts.push(...allocation.supplierParts);
  const northNeed = Array.from(total.cities.entries())
    .filter(([cityKey]) => cityKey !== "tyumen")
    .reduce((sum, [, quantity]) => sum + quantity, 0);
  const supplierDemandNeed = Number((tyumenSupplierNeed + allocation.supplierNorthNeed).toFixed(2));
  const supplierNeed = supplierUnitsFromPieces(supplierDemandNeed, supplierUnitSize) || 0;
  const actualSupplierOrder = defaultNorthActualSupplierOrder(summary, position, supplierNeed);

  return {
    key: total.key,
    baseKey: total.baseKey || position.baseKey || total.key,
    variant: total.variant || position.variant || "",
    variantLabel: total.variantLabel || position.variantLabel || "",
    articleRaw: total.articleRaw || position.articleRaw || "",
    name: total.name || position.name,
    unit: total.unit || position.unit || "шт",
    supplierUnitSize,
    totalQuantity: Number(total.totalQuantity.toFixed(2)),
    blankBoxSize: position.blankBoxSize,
    cities: northTotalCityParts(total),
    northNeed: Number(northNeed.toFixed(2)),
    budget: { ...tyumen?.budget, prices: position.budgetPrices || [] },
    tyumenOrder: tyumenUploadedOrder,
    tyumenPlannedOrder: Number(tyumenPlannedOrder.toFixed(2)),
    tyumenStock,
    tyumenInTransit,
    tyumenWarehouseStock: tyumen?.warehouseStock ?? null,
    tyumenWarehouseTransit: tyumen?.warehouseTransit ?? null,
    tyumenTarget,
    tyumenFree: Number(tyumenFree.toFixed(2)),
      fromTyumen: Number(allocation.fromTyumen.toFixed(2)),
      supplierNorthNeed: Number(allocation.supplierNorthNeed.toFixed(2)),
      supplierDemandNeed,
      supplierNeed,
    actualSupplierOrder,
    allowsRoundedShortSupplierOrder: summary.kind === "klapp",
    minimumExtra: actualSupplierOrder != null ? Number(Math.max(0, actualSupplierOrder - supplierNeed).toFixed(2)) : 0,
    supplierParts,
    tyumenParts: allocation.tyumenParts,
    hasTyumenSource: Boolean(tyumen),
    novacutanMinimum: position.novacutanMinimum ?? null,
  };
}

function buildNorthPlan(summary, totals, tyumenSourceWorkbook = null) {
  const availability = readNorthTyumenAvailability(tyumenSourceWorkbook, summary.kind);
  const positionByKey = new Map(summary.positions.map((position) => [position.key, position]));
  const planRows = [];

  for (const total of totals.values()) {
    const position = positionByKey.get(total.key) || {
      key: total.key,
      name: total.name,
      articleRaw: total.articleRaw,
      unit: total.unit,
    };
    const tyumen = northAvailabilityForTotal(availability, total, summary.kind);
    const tyumenTableOrder = tyumen
      ? (tyumen.hasOrderedFact ? (tyumen.orderedFact ?? 0) : roundHalfUp(Math.max(0, tyumen.recommended || 0)))
      : 0;
    planRows.push(northPlanRowFromTotal(summary, total, position, tyumen, tyumenTableOrder));
  }

  return planRows.sort((left, right) => {
    const supplierDiff = Number(right.supplierNeed > 0) - Number(left.supplierNeed > 0);
    if (supplierDiff) return supplierDiff;
    return left.name.localeCompare(right.name, "ru");
  });
}

function editedNorthTotals(result, editsByKey) {
  const totals = new Map();
  for (const row of result.planRows) {
    const edit = editsByKey.get(row.key);
    const cityQuantities = edit?.cities || Object.fromEntries((row.cities || []).map((city) => [city.key, city.quantity]));
    const cities = new Map();
    let totalQuantity = 0;
    for (const city of NORTH_CITIES) {
      const quantity = parseNumber(cityQuantities[city.key]) ?? 0;
      if (quantity <= 0) continue;
      cities.set(city.key, quantity);
      totalQuantity += quantity;
    }
    totals.set(row.key, {
      key: row.key,
      baseKey: row.baseKey || row.key,
      variant: row.variant || "",
      variantLabel: row.variantLabel || "",
      articleRaw: row.articleRaw,
      name: row.name,
      unit: row.unit,
      totalQuantity,
      cities,
    });
  }
  return totals;
}

function recalculateEditedNorthPlan(result, totals) {
  return result.planRows.map((row) => {
    const total = totals.get(row.key) || {
      key: row.key,
      articleRaw: row.articleRaw,
      name: row.name,
      unit: row.unit,
      totalQuantity: 0,
      cities: new Map(),
    };
    const tyumen = row.hasTyumenSource
      ? {
          stock: row.tyumenStock,
          inTransit: row.tyumenInTransit,
          targetStock: row.tyumenTarget,
          warehouseStock: row.tyumenWarehouseStock,
          warehouseTransit: row.tyumenWarehouseTransit,
        }
      : null;
    return { ...northPlanRowFromTotal(result.summary, total, row, tyumen, row.tyumenPlannedOrder), budgetComment: row.budgetComment };
  });
}

function actualQuantityByKey(planRows, editsByKey, options = {}) {
  const quantities = new Map();
  for (const row of planRows) {
    const edited = editsByKey.has(row.key) ? parseNumber(editsByKey.get(row.key).actualSupplierOrder) : null;
    const value = edited ?? row.actualSupplierOrder ?? 0;
    if (!options.allowShortSupplierOrder && Number(value || 0) < Number(row.supplierNeed || 0)) {
      throw new Error(`По позиции «${row.name}» факт у поставщика меньше общей нехватки Тюмени и северных городов.`);
    }
    quantities.set(row.key, Number(value || 0));
  }
  return quantities;
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

export function finalizeNorthOrderFiles(result, edits = [], options = {}) {
  const editsByKey = new Map(edits.map((edit) => [edit.key, edit]));
  const totals = editedNorthTotals(result, editsByKey);
  const planRows = recalculateEditedNorthPlan(result, totals);
  const actualByKey = actualQuantityByKey(planRows, editsByKey, options);
  const summaries = result.summaries?.length ? result.summaries : [result.summary];
  const summaryFiles = summaries.map((summary) => {
    const write = writeNorthSummaryWorkbook(summary, totals, actualByKey);
    return {
      workbook: budgetNorthWorkbookPricing(summary,result.budgetPricing),
      fileName: summary.summaryFileName || result.summaryFileName,
      label: summary.summaryLabel || "общий бланк",
      appended: write.appended,
      adjustedCount: write.adjustedCount,
    };
  });
  const adjustedCount = planRows.filter((row) => {
    const actual = Number(actualByKey.get(row.key) || 0);
    return actual > 0 && actual !== row.supplierNeed;
  }).length;

  return {
    summaryWorkbook: summaryFiles[0]?.workbook || result.summary.workbook,
    summaryFileName: summaryFiles[0]?.fileName || result.summaryFileName,
    summaryFiles,
    uploadedCities: result.uploadedCities,
    appendedToSummary: summaryFiles.flatMap((file) => file.appended),
    adjustedToMinimum: adjustedCount,
    totalsCount: Array.from(actualByKey.values()).filter((quantity) => quantity > 0).length,
    orderTable: novacutanNorthOrderTable(result.summary, planRows, actualByKey),
    transfers: transferFilesFromTotals(totals, result.transferCities),
  };
}

export function buildNorthOrderFiles(blanks, options = {}) {
  if (!Array.isArray(blanks) || !blanks.length) throw new Error("Загрузите хотя бы один бланк для раздела «Север».");
  if (options.tyumenSourceWorkbook) {
    validateNorthTyumenSourceWorkbook(options.tyumenSourceWorkbook, options.tyumenSourceFileName || "");
  }

  const northBrand = options.brand || "";
  const cityKeys = new Set();
  const prepared = blanks.map((blank, index) => {
    const city = detectNorthCity(blank.workbook, blank.fileName);
    const variant = blank.variant || "";
    const duplicateKey = `${city.key}:${variant || "default"}`;
    const duplicateLabel = variant ? `${city.label} ${blank.variantLabel || variant}` : city.label;
    if (cityKeys.has(duplicateKey)) throw new Error(`Загружено несколько бланков ${duplicateLabel}. Оставьте один файл на город и тип бланка.`);
    cityKeys.add(duplicateKey);
    const extracted = northPositions(blank.workbook, `north-${index}`, variant ? `${city.label} ${blank.variantLabel || variant}` : city.label, blank.fileName, northBrand);
    const positions = scopedNorthPositions(extracted.positions, variant, blank.variantLabel || "");
    return { ...blank, city, ...extracted, positions, variant, variantLabel: blank.variantLabel || "" };
  });

  const totals = new Map();
  for (const file of prepared) {
    for (const item of file.positions) addNorthTotal(totals, item, file.city);
  }

  const hasVariants = prepared.some((file) => file.variant);
  const summaries = hasVariants
    ? Array.from(new Map(prepared.map((file) => [file.variant, file])).entries())
      .map(([variant, firstFile]) => {
        const group = prepared.filter((file) => file.variant === variant);
        const summary = group.find((file) => file.city.key === "tyumen") || firstFile;
        return {
          ...summary,
          positions: summary.positions,
          summaryFileName: `Север общий бланк ${summary.variantLabel || variant}.${outputExtension(summary.fileName)}`,
          summaryLabel: `общий бланк ${summary.variantLabel || variant}`,
        };
      })
    : [];
  const summary = summaries[0] || prepared.find((file) => file.city.key === "tyumen") || prepared[0];
  if (!summary.summaryFileName) summary.summaryFileName = `Север общий бланк.${outputExtension(summary.fileName)}`;
  if (!summary.summaryLabel) summary.summaryLabel = "общий бланк";
  // Zero-demand items may participate in budget top-up, but never create city transfers.
  const allPositions = prepared.flatMap(file => file.positions);
  if (options.tyumenSourceWorkbook) {
    const available = readNorthTyumenAvailability(options.tyumenSourceWorkbook, summary.kind);
    for (const p of allPositions) {
      if (totals.has(p.key) || !northAvailabilityForTotal(available, p, summary.kind)) continue;
      totals.set(p.key, { ...p, totalQuantity: 0, cities: new Map([['tyumen', 0]]) });
    }
  }
  const planRows = buildNorthPlan({ ...summary, positions: allPositions }, totals, options.tyumenSourceWorkbook || null);
  const transferCityMap = new Map(prepared.filter((file) => file.city.key !== "tyumen").map((file) => [file.city.key, file.city]));
  const transferCities = Array.from(transferCityMap.values());
  const confirmationGroups = Array.from(new Map(prepared.map((file) => [file.city.key, file.city])).values()).map((city) => ({
    city,
    variants: prepared
      .filter((file) => file.city.key === city.key)
      .map((file) => file.variantLabel || file.fileName),
  }));

  return {
    summary,
    summaries: summaries.length ? summaries : [summary],
    totals,
    summaryFileName: summary.summaryFileName,
    uploadedCities: Array.from(new Set(prepared.map((file) => file.city.label))),
    confirmationGroups,
    planRows,
    hasTyumenSource: Boolean(options.tyumenSourceWorkbook),
    transferCities,
    transfers: transferFilesFromTotals(totals, transferCities),
  };
}
