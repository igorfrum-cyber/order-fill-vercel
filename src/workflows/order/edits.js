import { blankDetectionOptions, brandRule } from "./../../domain/brands/rules.js";
import { asText } from "./../../shared/text.js";
import { detectColumns } from "./../../infrastructure/excel/columns.js";
import { setNumericCell, setTextCell } from "./../../infrastructure/excel/xml.js";
import { detectKlappBlank, detectNovacutanBlank, detectSothysVariantBlank } from "./../../infrastructure/excel/blanks.js";

export function normalizedBaselineQuantity(rowInfo, rule = brandRule("angiopharm")) {
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
