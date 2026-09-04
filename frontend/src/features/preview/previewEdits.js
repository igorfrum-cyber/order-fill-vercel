import { rowKey } from "../order/reviewEdits.js";
import { quantityDisplay } from "../report/rowPresentation.js";
import { previewFileTitle } from "./fileTitle.js";

export function isSourcePreviewFile(file) {
  return previewFileTitle(file) === "Таблица 1С";
}

export function blankIdForPreviewFile(files = [], fileId) {
  const blanks = files.filter((file) => !isSourcePreviewFile(file));
  const index = blanks.findIndex((file) => file.id === fileId);
  if (index < 0) return "";
  return `blank-${index + 1}`;
}

export function defaultPreviewFileId(files = []) {
  const source = files.find((file) => isSourcePreviewFile(file));
  return source?.id || files[0]?.id || "";
}

export function orderSheetIndex(sheets = []) {
  const withFact = sheets.find((sheet) => Number(sheet?.quantity_column) > 0);
  if (withFact) return withFact.index;
  const withHeader = sheets.find((sheet) => Number(sheet?.header_row) > 0);
  return withHeader?.index ?? sheets[0]?.index ?? 0;
}

export function needsHeaderScan(sheet, { sourceFile, jobId, fileId } = {}) {
  if (!sourceFile || !jobId || !fileId || !sheet) return false;
  const headerRow = Number(sheet.header_row);
  if (!Number.isFinite(headerRow) || headerRow < 1) return false;
  return !(Number(sheet.quantity_column) > 0);
}

export function findEditColumns(headerCells = []) {
  let quantity = 0;
  let comment = 0;
  for (let index = 0; index < headerCells.length; index += 1) {
    const folded = String(headerCells[index] || "").toLowerCase().trim();
    if (folded.includes("заказано") && folded.includes("факт")) quantity = index + 1;
    if (folded === "комментарий" || folded.startsWith("комментарий")) comment = index + 1;
  }
  return { quantity, comment };
}

export function previewOverlays(rows = [], edits = new Map(), { files = [], fileId, quantityColumn, commentColumn } = {}) {
  if (isSourcePreviewFile(files.find((file) => file.id === fileId))) {
    return sourceOverlays(rows, edits, { quantityColumn, commentColumn });
  }
  return blankOverlays(rows, edits, { files, fileId });
}

function blankOverlays(rows, edits, { files, fileId }) {
  const overlays = new Map();
  const blankId = blankIdForPreviewFile(files, fileId);
  if (!blankId) return overlays;

  for (const row of rows) {
    if (row.editable === false) continue;
    const rowBlankId = row.blankId && row.blankId !== "main" ? row.blankId : "blank-1";
    if (rowBlankId !== blankId) continue;
    const sheetRow = Number(row.blankRow);
    const column = Number(row.blankQuantityCol);
    if (!Number.isFinite(sheetRow) || sheetRow < 1 || !Number.isFinite(column) || column < 1) continue;
    const key = rowKey(row);
    const edit = edits.get(key) || { value: row.inserted ?? "", comment: "" };
    overlays.set(`${sheetRow}:${column}`, {
      field: "value",
      value: quantityDisplay(edit.value),
    });
  }
  return overlays;
}

function sourceOverlays(rows, edits, { quantityColumn, commentColumn } = {}) {
  const overlays = new Map();
  const factCol = Number(quantityColumn);
  const commentCol = Number(commentColumn);
  if (!Number.isFinite(factCol) || factCol < 1) return overlays;

  for (const row of rows) {
    if (row.editable === false) continue;
    const sheetRow = Number(row.sourceRow);
    if (!Number.isFinite(sheetRow) || sheetRow < 1) continue;
    const key = rowKey(row);
    const edit = edits.get(key) || { value: row.inserted ?? "", comment: "" };
    overlays.set(`${sheetRow}:${factCol}`, {
      key,
      field: "value",
      value: quantityDisplay(edit.value),
      comment: Number.isFinite(commentCol) && commentCol > 0 ? "" : String(edit.comment || "").trim(),
    });
    if (Number.isFinite(commentCol) && commentCol > 0) {
      overlays.set(`${sheetRow}:${commentCol}`, {
        key,
        field: "comment",
        value: String(edit.comment || "").trim(),
      });
    }
  }
  return overlays;
}

export function mergePreviewOverlays(derived = new Map(), quantity = new Map()) {
  const overlays = new Map(derived);
  for (const [key, value] of quantity) {
    overlays.set(key, value);
  }
  return overlays;
}

export function isQuantityOverlay(overlay) {
  return Boolean(overlay && overlay.field === "value" && overlay.key);
}

export function isCommentOverlay(overlay) {
  return Boolean(overlay && overlay.field === "comment" && overlay.key);
}

export function needsEditResubmit({ finalized, dirty, hasDeviations } = {}) {
  return Boolean(hasDeviations && (!finalized || dirty));
}
