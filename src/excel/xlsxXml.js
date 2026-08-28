import { strFromU8, strToU8, unzipSync, zipSync } from "fflate";
import { DOMParser, XMLSerializer } from "@xmldom/xmldom";
import { asText } from "../domain/normalization.js";

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
  forceFormulaRecalculation(files);
  return zipSync(files, { level: 6 });
}

export function forceFormulaRecalculation(files) {
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

export function parseXml(bytes) {
  return XML_PARSER.parseFromString(strFromU8(bytes), "application/xml");
}

function normalizeWorkbookTarget(target) {
  const clean = target.replace(/^\/+/, "");
  return clean.startsWith("xl/") ? clean : `xl/${clean}`;
}

export function elements(root, tagName) {
  return Array.from(root.getElementsByTagName(tagName));
}

export function firstElement(parent, tagName) {
  return parent.getElementsByTagName(tagName)[0] || null;
}

function parseSharedStrings(xml) {
  return elements(xml, "si").map((si) => elements(si, "t").map((t) => t.textContent || "").join(""));
}

export function readSheetCells(xml, sharedStrings) {
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

export function cellKey(row, col) {
  return `${row}:${col}`;
}

export function parseCellRef(ref) {
  const match = /^([A-Z]+)(\d+)$/.exec(ref);
  if (!match) throw new Error(`Некорректная ссылка ячейки: ${ref}`);
  return { col: columnNameToNumber(match[1]), row: Number(match[2]) };
}

function columnNameToNumber(name) {
  let result = 0;
  for (const ch of name) result = result * 26 + ch.charCodeAt(0) - 64;
  return result;
}

export function columnNumberToName(number) {
  let result = "";
  let current = number;
  while (current > 0) {
    const mod = (current - 1) % 26;
    result = String.fromCharCode(65 + mod) + result;
    current = Math.floor((current - 1) / 26);
  }
  return result;
}

export function sheetBounds(sheet) {
  let maxRow = 0;
  let maxColumn = 0;
  for (const cell of sheet.cells.values()) {
    maxRow = Math.max(maxRow, cell.row);
    maxColumn = Math.max(maxColumn, cell.col);
  }
  return { maxRow, maxColumn };
}

export function sheetCellValue(sheet, row, col) {
  return sheet.cells.get(cellKey(row, col))?.value ?? "";
}

export function refreshCellValue(sheet, row, col) {
  const cell = sheet.cells.get(cellKey(row, col));
  if (cell) cell.value = readCellValue(cell.node, []);
}

export function findOrCreateCell(sheet, rowNumber, colNumber) {
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

export function setNumericCell(sheet, row, col, value) {
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

export function setTextCell(sheet, row, col, value) {
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
