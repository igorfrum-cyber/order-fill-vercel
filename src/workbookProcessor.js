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
  return zipSync(files, { level: 6 });
}

function parseXml(bytes) {
  return XML_PARSER.parseFromString(strFromU8(bytes), "application/xml");
}

function normalizeWorkbookTarget(target) {
  const clean = target.replace(/^\/+/, "");
  return clean.startsWith("xl/") ? clean : `xl/${clean}`;
}

function elements(root, tagName) {
  return Array.from(root.getElementsByTagName(tagName));
}

function firstElement(parent, tagName) {
  return parent.getElementsByTagName(tagName)[0] || null;
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
  return String(value).replace(/\n/g, " ").trim();
}

export function normalizeHeader(value) {
  return asText(value).toLowerCase().replaceAll("ё", "е").replace(/[^\p{L}\p{N}%]+/gu, " ").replace(/\s+/g, " ").trim();
}

export function normalizeArticle(value) {
  return asText(value).replace(/[АВЕКМНОРСТХУавекмнорстху]/g, (ch) => ARTICLE_TRANSLATION.get(ch) || ch).toUpperCase().replace(/[^A-Z0-9]/g, "");
}

export function normalizeName(value) {
  return normalizeHeader(value).replace(/\bан\b/g, " ").replace(/\bangiopharm\b/g, " ").replace(/\s+/g, " ").trim();
}

export function parseNumber(value) {
  if (value == null || asText(value) === "") return null;
  const number = Number(asText(value).replace(/\s+/g, "").replace(",", "."));
  return Number.isFinite(number) ? number : null;
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
  };
}

function blankMatchers() {
  return {
    article: (h) => h.includes("арт") || h.includes("артикул") || h.includes("код"),
    name: (h) => h.includes("товар") || h.includes("номенклатура") || h.includes("наименование") || h.includes("название"),
    quantity: (h) => h.includes("кол во") || h.includes("количество") || h.includes("кол-во") || h.includes("к во") || h.includes("qty"),
  };
}

function combinations(arrays) {
  return arrays.reduce((acc, current) => acc.flatMap((items) => current.map((item) => [...items, item])), [[]]);
}

export function detectColumns(workbook, mode) {
  const matchers = mode === "source" ? sourceMatchers() : blankMatchers();
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

function readSource(workbook, orderMonth) {
  const periodInfo = validateSourcePeriods(workbook, orderMonth);
  const detection = detectColumns(workbook, "source");
  const items = [];
  const { maxRow } = sheetBounds(detection.sheet);
  for (let row = detection.headerRow + 1; row <= maxRow; row += 1) {
    const articleRaw = asText(sheetCellValue(detection.sheet, row, detection.columns.article));
    const name = asText(sheetCellValue(detection.sheet, row, detection.columns.name));
    const recommended = parseNumber(sheetCellValue(detection.sheet, row, detection.columns.recommended));
    if (!articleRaw && !name && recommended == null) continue;
    if (recommended == null) continue;
    items.push({ rowIndex: row, articleRaw, article: normalizeArticle(articleRaw), name, recommended, rounded: roundHalfUp(recommended) });
  }
  return { detection, items, periodInfo };
}

function chooseCandidate(candidates, blankName) {
  return candidates.map((item) => ({ item, score: similarity(blankName, item.name) })).sort((left, right) => right.score - left.score)[0];
}

function chooseNameFallback(candidates, blankName) {
  const scored = candidates.filter((item) => !item.article).map((item) => ({ item, score: similarity(blankName, item.name) })).sort((left, right) => right.score - left.score);
  if (!scored.length) return { item: null, score: 0 };
  const bestNonpositive = scored.find((entry) => entry.item.rounded <= 0 && entry.score >= 0.72);
  if (bestNonpositive) return bestNonpositive;
  if (scored.length > 1 && scored[0].score - scored[1].score < 0.08) return { item: null, score: scored[0].score };
  if (scored[0].score < 0.72) return { item: null, score: scored[0].score };
  return scored[0];
}

export function fillWorkbook({ sourceWorkbook, blankWorkbook, orderMonth }) {
  const source = readSource(sourceWorkbook, orderMonth);
  const sourceIndex = new Map();
  const noArticleItems = [];
  for (const item of source.items) {
    if (item.article) {
      if (!sourceIndex.has(item.article)) sourceIndex.set(item.article, []);
      sourceIndex.get(item.article).push(item);
    } else {
      noArticleItems.push(item);
    }
  }
  const blank = detectColumns(blankWorkbook, "blank");
  const reportRows = [];
  let filled = 0;
  let leftBlank = 0;
  let suspicious = 0;
  let unmatched = 0;
  let duplicates = 0;
  const { maxRow } = sheetBounds(blank.sheet);
  for (let row = blank.headerRow + 1; row <= maxRow; row += 1) {
    const blankArticleRaw = asText(sheetCellValue(blank.sheet, row, blank.columns.article));
    const blankArticle = normalizeArticle(blankArticleRaw);
    const blankName = asText(sheetCellValue(blank.sheet, row, blank.columns.name));
    if (!blankArticle) continue;
    let selected;
    let score;
    let status;
    const candidates = sourceIndex.get(blankArticle) || [];
    if (!candidates.length) {
      const fallback = chooseNameFallback(noArticleItems, blankName);
      if (!fallback.item) {
        unmatched += 1;
        continue;
      }
      selected = fallback.item;
      score = fallback.score;
      if (selected.rounded > 0) {
        suspicious += 1;
        reportRows.push(makeReportRow("warning_name_only", row, blankArticleRaw, blankName, selected, score));
        continue;
      }
      status = "matched_by_name";
    } else {
      if (candidates.length > 1) duplicates += 1;
      const candidate = chooseCandidate(candidates, blankName);
      selected = candidate.item;
      score = candidate.score;
      status = "matched";
      if (score < 0.32) {
        status = "warning_name_differs";
        suspicious += 1;
      }
    }
    if (selected.rounded <= 0) {
      setNumericCell(blank.sheet, row, blank.columns.quantity, null);
      leftBlank += 1;
      status = "left_blank_nonpositive";
    } else {
      setNumericCell(blank.sheet, row, blank.columns.quantity, selected.rounded);
      filled += 1;
    }
    reportRows.push(makeReportRow(status, row, blankArticleRaw, blankName, selected, score));
  }
  return {
    blankWorkbook,
    blankDetection: blank,
    summary: {
      filled,
      leftBlank,
      suspicious,
      unmatched,
      duplicates,
      sourceItems: source.items.length,
      sourceArticles: sourceIndex.size,
      sourceSheet: source.detection.sheetName,
      sourceHeaderRow: source.detection.headerRow,
      blankSheet: blank.sheetName,
      blankHeaderRow: blank.headerRow,
      ...source.periodInfo,
    },
    reportRows,
  };
}

function makeReportRow(status, row, blankArticle, blankName, selected, score) {
  return {
    status,
    blankRow: row,
    blankArticle,
    blankName,
    sourceRow: selected.rowIndex,
    sourceArticle: selected.articleRaw,
    sourceName: selected.name,
    recommended: selected.recommended,
    rounded: selected.rounded,
    similarity: Number(score.toFixed(4)),
  };
}

function findOrCreateCell(sheet, rowNumber, colNumber) {
  const key = cellKey(rowNumber, colNumber);
  const existing = sheet.cells.get(key);
  if (existing) return existing.node;
  const sheetData = firstElement(sheet.xml, "sheetData");
  let row = Array.from(sheetData.getElementsByTagName("row")).find((node) => Number(node.getAttribute("r")) === rowNumber);
  if (!row) {
    row = sheet.xml.createElementNS(NS_MAIN, "row");
    row.setAttribute("r", String(rowNumber));
    sheetData.appendChild(row);
  }
  const ref = `${columnNumberToName(colNumber)}${rowNumber}`;
  const cell = sheet.xml.createElementNS(NS_MAIN, "c");
  cell.setAttribute("r", ref);
  const cells = Array.from(row.getElementsByTagName("c"));
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

export function outputFileName(originalName) {
  const stem = asText(originalName).replace(/\.(xlsx|xlsm)$/i, "").replace(/[^\p{L}\p{N}_ .-]+/gu, "").trim() || "blank";
  return `${stem} заполненный.xlsx`;
}
