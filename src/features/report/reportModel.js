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

export function normalizeReportRow(row) {
  return {
    ...row,
    blankId: row.blank_id || row.blankId || "main",
    blankLabel: row.blank_label || row.blankLabel || "",
    blankRow: row.blank_row || row.blankRow || "",
    blankQuantityCol: row.blank_quantity_col || row.blankQuantityCol || null,
    blankArticle: row.blank_article || row.blankArticle || "",
    blankName: row.blank_name || row.blankName || "",
    blankUnit: row.blank_unit || row.blankUnit || "",
    blankBoxSize: row.blank_box_size || row.blankBoxSize || "",
    sourceRow: row.source_row || row.sourceRow || null,
    sourceArticle: row.source_article || row.sourceArticle || "",
    sourceName: row.source_name || row.sourceName || "",
    hasOrderedFact: row.has_ordered_fact || row.hasOrderedFact || false,
    orderedFact: row.ordered_fact ?? row.orderedFact ?? null,
    sourceComment: row.source_comment || row.sourceComment || "",
    inTransit: row.in_transit ?? row.inTransit ?? "",
    baseRounded: row.base_rounded ?? row.baseRounded ?? null,
    autoComment: row.auto_comment || row.autoComment || "",
    boxAdjusted: row.box_adjusted || row.boxAdjusted || false,
    duplicateCandidates: row.duplicate_candidates || row.duplicateCandidates || [],
    editable: row.editable !== false,
  };
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
    duplicates: rows.filter((row) => row.duplicate).length,
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
