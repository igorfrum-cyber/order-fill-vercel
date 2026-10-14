import { DOMParser, XMLSerializer } from "@xmldom/xmldom";
import { strFromU8 } from "fflate";
import { asText } from "./../../shared/text.js";

export const NS_MAIN = "http://schemas.openxmlformats.org/spreadsheetml/2006/main";

export const XML_PARSER = new DOMParser();

export const XML_SERIALIZER = new XMLSerializer();

export function parseXml(bytes) {
  return XML_PARSER.parseFromString(strFromU8(bytes), "application/xml");
}

export function elements(root, tagName) {
  // Spreadsheet XML may use a default namespace or prefixes such as s:sheet.
  return Array.from(root.getElementsByTagNameNS("*", tagName));
}

export function firstElement(parent, tagName) {
  return parent.getElementsByTagNameNS("*", tagName)[0] || null;
}

export function readCellValue(cell, sharedStrings) {
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

export function columnNameToNumber(name) {
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

export function clearCellChildren(cell) {
  while (cell.firstChild) cell.removeChild(cell.firstChild);
  cell.removeAttribute("t");
}

export function copyCellStyle(sheet, row, fromCol, toCol) {
  const source = sheet.cells.get(cellKey(row, fromCol))?.node;
  if (!source) return;
  const target = findOrCreateCell(sheet, row, toCol);
  const style = source.getAttribute("s");
  if (style != null) target.setAttribute("s", style);
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
