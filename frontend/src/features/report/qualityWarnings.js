import { isIssueRow } from "./reportModel.js";
import { isManualDeviation } from "../order/reviewEdits.js";

export function qualityWarningSummary({ rows, results = [], edits = new Map() }) {
  const issueCount = rows.filter(isIssueRow).length;
  const duplicateCount = rows.filter((row) => row.duplicate).length;
  const notInSourceCount = rows.filter((row) => row.status === "not_in_source").length;
  const notInBlankCount = rows.filter((row) => row.status === "not_in_blank").length;
  const manualCount = rows.filter((row) => isManualDeviation(row, edits)).length;
  const blankDuplicateCount = results.reduce((sum, result) => sum + Number(result.summary?.blankDuplicateArticles || 0), 0);
  const total = issueCount + duplicateCount + notInSourceCount + notInBlankCount + manualCount + blankDuplicateCount;
  return {
    issueCount,
    duplicateCount,
    notInSourceCount,
    notInBlankCount,
    manualCount,
    blankDuplicateCount,
    total,
  };
}

export function qualityWarningLines(summary) {
  if (!summary.total) return [];
  return [
    `Проверьте ${summary.total} спорных строк/ситуаций перед скачиванием.`,
    summary.issueCount ? `Проверить: ${summary.issueCount}` : "",
    summary.duplicateCount ? `Дубли: ${summary.duplicateCount}` : "",
    summary.notInSourceCount ? `Нет в таблице: ${summary.notInSourceCount}` : "",
    summary.notInBlankCount ? `Нет в бланке: ${summary.notInBlankCount}` : "",
    summary.manualCount ? `Ручные отклонения: ${summary.manualCount}` : "",
    summary.blankDuplicateCount ? `Дубли артикулов в бланке: ${summary.blankDuplicateCount}` : "",
  ].filter(Boolean);
}

export function issueReportRows(rows) {
  return rows.filter((row) => isIssueRow(row) || row.status === "not_in_source" || row.duplicate);
}
