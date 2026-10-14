import { normalizeOrderValue } from "./../../workflows/order/edits.js";

export function createOrderEdits(context) {
  const {elements, state, actions} = context;
function collectEdits() {
  return state.currentReportRows
    .filter((row) => row.editable !== false)
    .map((row) => {
      const key = row.key || `${row.blankId}:${row.blankRow}`;
      const edit = actions.rowEdit(row);
      return {
        key,
        blankId: row.blankId,
        blankRow: Number(row.blankRow),
        value: edit.value,
        comment: edit.comment,
      };
    });
}

function validateEdits() {
  let invalidCount = 0;
  for (const row of elements.reportBody.querySelectorAll("tr")) row.classList.remove("invalid");
  for (const row of elements.priorityBody.querySelectorAll("tr")) row.classList.remove("invalid");

  for (const rowInfo of state.currentReportRows) {
    if (rowInfo.editable === false) continue;
    const key = rowInfo.key || `${rowInfo.blankId}:${rowInfo.blankRow}`;
    const edit = actions.rowEdit(rowInfo);
    const initial = rowInfo.inserted == null ? null : Number(rowInfo.inserted);
    const baseline = actions.baselineForReportRow(rowInfo);
    const autoComment = (rowInfo.autoComment || "").trim().toLowerCase();
    let value;
    try {
      value = normalizeOrderValue(edit.value);
    } catch {
      document.querySelectorAll(`[data-key="${CSS.escape(key)}"]`).forEach((input) => input.closest("tr")?.classList.add("invalid"));
      invalidCount += 1;
      continue;
    }
    const comment = edit.comment.trim();
    const requiresComment = value !== baseline;
    const stillAutoComment = autoComment && comment.toLowerCase() === autoComment;
    const autoCommentAllowed = stillAutoComment && value === initial;
    if (requiresComment && (!comment || (stillAutoComment && !autoCommentAllowed))) {
      document.querySelectorAll(`[data-key="${CSS.escape(key)}"]`).forEach((input) => input.closest("tr")?.classList.add("invalid"));
      invalidCount += 1;
    }
  }

  if (invalidCount > 0) {
    const firstInvalid = elements.priorityBody.querySelector("tr.invalid") || elements.reportBody.querySelector("tr.invalid");
    firstInvalid?.scrollIntoView({ block: "center", behavior: "smooth" });
    alert("Есть строки, где изменено значение «Вставлено», но не заполнен новый комментарий.");
    return false;
  }
  return true;
}

function isManualDeviation(rowInfo) {
  if (rowInfo.editable === false) return false;
  const edit = actions.rowEdit(rowInfo);
  let value;
  try {
    value = normalizeOrderValue(edit.value);
  } catch {
    return true;
  }
  const baseline = actions.baselineForReportRow(rowInfo);
  return value !== baseline;
}

function confirmQualityWarnings() {
  const issueCount = state.currentReportRows.filter((row) => row.status === "warning_name_differs" || row.status === "warning_name_only").length;
  const duplicateCount = state.currentReportRows.filter((row) => row.duplicate).length;
  const notInSourceCount = state.currentReportRows.filter((row) => row.status === "not_in_source").length;
  const notInBlankCount = state.currentReportRows.filter((row) => row.status === "not_in_blank").length;
  const manualCount = state.currentReportRows.filter(isManualDeviation).length;
  const blankDuplicateCount = state.currentResults.reduce((sum, result) => sum + (result.summary.blankDuplicateArticles || 0), 0);
  const total = issueCount + duplicateCount + notInSourceCount + notInBlankCount + manualCount + blankDuplicateCount;
  if (!total) return true;

  const lines = [
    `Проверьте ${total} спорных строк/ситуаций перед скачиванием.`,
    issueCount ? `Проверить: ${issueCount}` : "",
    duplicateCount ? `Дубли: ${duplicateCount}` : "",
    notInSourceCount ? `Нет в таблице: ${notInSourceCount}` : "",
    notInBlankCount ? `Нет в бланке: ${notInBlankCount}` : "",
    manualCount ? `Ручные отклонения: ${manualCount}` : "",
    blankDuplicateCount ? `Дубли артикулов в бланке: ${blankDuplicateCount}` : "",
    "",
    "Продолжить скачивание?",
  ].filter((line) => line !== "").join("\n");
  return window.confirm(lines);
}

function rowNeedsComment(row) {
  const qtyInput = row.querySelector(".qty-input");
  const commentInput = row.querySelector(".comment-input");
  if (!qtyInput || !commentInput) return false;

  const initial = qtyInput.dataset.initialValue === "" ? null : Number(qtyInput.dataset.initialValue);
  const baseline = qtyInput.dataset.baselineValue === "" ? null : Number(qtyInput.dataset.baselineValue);
  const autoComment = (qtyInput.dataset.autoComment || "").trim().toLowerCase();
  let value;
  try {
    value = normalizeOrderValue(qtyInput.value);
  } catch {
    return false;
  }

  const comment = commentInput.value.trim();
  const stillAutoComment = autoComment && comment.toLowerCase() === autoComment;
  const autoCommentAllowed = stillAutoComment && value === initial;
  return value !== baseline && (!comment || (stillAutoComment && !autoCommentAllowed));
}

function updateCommentHint(row) {
  const commentInput = row.querySelector(".comment-input");
  if (!commentInput) return;
  commentInput.classList.toggle("needs-comment", rowNeedsComment(row));
}
  return { collectEdits, validateEdits, isManualDeviation, confirmQualityWarnings, rowNeedsComment, updateCommentHint };
}
