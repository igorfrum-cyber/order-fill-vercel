export function statusLabel(status) {
  const labels = {
    matched: "Заполнено",
    matched_by_name: "По названию",
    warning_name_differs: "Проверить название",
    warning_name_only: "Проверить без артикула",
    left_blank_nonpositive: "Пусто",
    not_in_source: "Нет в таблице",
    not_in_blank: "Нет в бланке",
    source_duplicate: "Дубль в таблице",
  };
  return labels[status] || status;
}

export function jobStatusText(job) {
  const labels = {
    queued: "Задача в очереди...",
    processing: "Обработка...",
    needs_review: "Проверьте расчет",
    finalizing: "Готовлю файлы...",
    completed: "Готово",
    failed: job.error?.message || "Ошибка обработки",
  };
  return labels[job.status] || "Обработка...";
}

export function isIssueRow(row) {
  return row.status === "warning_name_differs" || row.status === "warning_name_only";
}

export function initialComment(row) {
  return row.sourceComment || row.autoComment || "";
}

export function baselineForReportRow(row) {
  if (Number(row.recommended) < 1.5 || Number(row.rounded) <= 0) return null;
  return Number(row.rounded);
}

export function reportSummaryFromRows(rows, job, fallback = {}) {
  return {
    brand: job.brand || fallback.brand || "",
    orderMonthLabel: job.order_month || fallback.orderMonth || "",
    actualMainPeriod: "",
    actualPreviousPeriod: "",
    sourceCity: "",
    cityRule: "",
    deliveryWeeks: "",
    blankDuplicateArticles: 0,
    filled: rows.filter((row) => (row.status === "matched" || row.status === "matched_by_name") && row.inserted != null).length,
    leftBlank: rows.filter((row) => row.status === "left_blank_nonpositive").length,
    suspicious: rows.filter(isIssueRow).length,
    unmatched: rows.filter((row) => row.status === "not_in_source").length,
  };
}

export function combinedSummary(results, reportRows = []) {
  const first = results[0]?.summary || {};
  const rows = reportRows.length ? reportRows : results.flatMap((result) => result.reportRows || []);
  return {
    ...first,
    filled: results.reduce((sum, result) => sum + Number(result.summary?.filled || 0), 0),
    leftBlank: results.reduce((sum, result) => sum + Number(result.summary?.leftBlank || 0), 0),
    suspicious: results.reduce((sum, result) => sum + Number(result.summary?.suspicious || 0), 0),
    notInSource: results.reduce((sum, result) => sum + Number(result.summary?.unmatched || 0), 0),
    notInBlank: rows.filter((row) => row.status === "not_in_blank").length,
    duplicates: first.duplicates != null
      ? Number(first.duplicates)
      : rows.filter((row) => row.duplicate && row.status !== "source_duplicate").length,
    blankDuplicateArticles: results.reduce((sum, result) => sum + Number(result.summary?.blankDuplicateArticles || 0), 0),
    blankWarnings: results.flatMap((result) => result.summary?.blankWarnings || []),
  };
}

export function rowMatchesFilter(row, filter) {
  if (!filter) return true;
  if (filter === "filled") return (row.status === "matched" || row.status === "matched_by_name") && row.inserted != null;
  if (filter === "leftBlank") return row.status === "left_blank_nonpositive";
  if (filter === "suspicious") return isIssueRow(row);
  if (filter === "notInSource") return row.status === "not_in_source";
  if (filter === "notInBlank") return row.status === "not_in_blank";
  if (filter === "duplicates") return Boolean(row.duplicate);
  return true;
}

export function isPriorityRow(row) {
  return isIssueRow(row) || Boolean(row.duplicate);
}
