import { baselineForReportRow, initialComment } from "../report/reportModel.js";
import { editRequiresComment, normalizeOrderValue } from "./editRules.js";

export function rowKey(row) {
  return row.key || `${row.blankId}:${row.blankRow}`;
}

export function editForRow(row, edits) {
  const key = rowKey(row);
  if (!edits.has(key)) {
    edits.set(key, { value: row.inserted ?? "", comment: initialComment(row) });
  }
  return edits.get(key);
}

export function initialEditState(rows) {
  return new Map(rows.map((row) => [rowKey(row), { value: row.inserted ?? "", comment: initialComment(row) }]));
}

export function validateReviewEdits(rows, edits) {
  const invalid = [];
  for (const row of rows) {
    if (row.editable === false) continue;
    const key = rowKey(row);
    const edit = editForRow(row, edits);
    let value;
    try {
      value = normalizeOrderValue(edit.value);
    } catch {
      invalid.push(key);
      continue;
    }
    const initial = row.inserted == null ? null : Number(row.inserted);
    if (editRequiresComment({
      value,
      baseline: baselineForReportRow(row),
      initial,
      comment: edit.comment,
      autoComment: row.autoComment,
    })) {
      invalid.push(key);
    }
  }
  return invalid;
}

export function collectReviewEdits(rows, edits) {
  return rows
    .filter((row) => row.editable !== false)
    .map((row) => {
      const edit = editForRow(row, edits);
      return {
        key: rowKey(row),
        blankId: row.blankId,
        blankRow: Number(row.blankRow),
        value: edit.value,
        comment: edit.comment,
      };
    });
}

export function isManualDeviation(row, edits) {
  if (row.editable === false) return false;
  const edit = editForRow(row, edits);
  let value;
  try {
    value = normalizeOrderValue(edit.value);
  } catch {
    return true;
  }
  return value !== baselineForReportRow(row);
}

export function hasManualDeviations(rows, edits) {
  return rows.some((row) => isManualDeviation(row, edits));
}
