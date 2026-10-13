import { strToU8, unzipSync, zipSync } from "fflate";
import { NS_MAIN, XML_SERIALIZER, cellKey, elements, firstElement, parseCellRef, parseXml, readCellValue } from "./xml.js";

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

export function removeNodesByAttribute(files, path, tagName, attrName, attrValuePart) {
  if (!files[path]) return;
  const xml = parseXml(files[path]);
  for (const node of elements(xml, tagName)) {
    if ((node.getAttribute(attrName) || "").includes(attrValuePart)) {
      node.parentNode?.removeChild(node);
    }
  }
  files[path] = strToU8(XML_SERIALIZER.serializeToString(xml));
}

export function normalizeWorkbookTarget(target) {
  const clean = target.replace(/^\/+/, "");
  return clean.startsWith("xl/") ? clean : `xl/${clean}`;
}

export function parseSharedStrings(xml) {
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
