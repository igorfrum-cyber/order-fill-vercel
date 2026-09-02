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
  return labels[status] || "Другое";
}

const JOB_STATUS_LABELS = {
  queued: "В очереди",
  processing: "Обработка",
  needs_review: "На проверке",
  finalizing: "Готовлю файлы",
  completed: "Готово",
  failed: "Ошибка",
};

export function jobStatusLabel(status) {
  return JOB_STATUS_LABELS[status] || "В работе";
}

const JOB_STATUS_HINTS = {
  queued: "Файлы ждут обработки.",
  processing: "Сервис читает файлы и считает количества.",
  needs_review: "Откройте выгрузку и проверьте строки.",
  finalizing: "Собираю готовые Excel-файлы.",
  completed: "Можно скачать готовые файлы.",
  failed: "Не получилось обработать. Откройте строку, чтобы увидеть причину.",
};

export function jobStatusHint(status) {
  return JOB_STATUS_HINTS[status] || "";
}

const LIVE_JOB_STATUSES = new Set(["queued", "processing", "needs_review", "finalizing"]);

export function liveJobs(jobs = []) {
  return (jobs || []).filter((job) => LIVE_JOB_STATUSES.has(job.status));
}

export function jobsEmptyState(role) {
  if (role === "platform_admin") {
    return "Пока нет выгрузок по выбранной компании.";
  }
  return "Пока нет выгрузок. Начните с бланка закупки или объединения Севера.";
}

export function jobStatusText(job) {
  const live = job?.progress_message || job?.progressMessage;
  if (live) return live;
  if (job?.status === "failed") return job.error?.message || JOB_STATUS_LABELS.failed;
  const liveLabels = {
    queued: "Задача в очереди...",
    processing: "Обработка...",
    needs_review: "Проверьте расчёт",
    finalizing: "Готовлю файлы...",
    completed: "Готово",
  };
  return liveLabels[job?.status] || jobStatusLabel(job?.status);
}

export function jobProgress(job) {
  if (job?.status === "needs_review" || job?.status === "completed" || job?.status === "failed") {
    return 1;
  }
  const raw = Number(job?.progress);
  const hasMessage = Boolean(job?.progress_message || job?.progressMessage);
  if (Number.isFinite(raw) && raw >= 0 && (raw > 0 || hasMessage)) {
    return Math.max(0, Math.min(1, raw));
  }
  const fallback = {
    queued: 0.05,
    processing: 0.12,
    finalizing: 0.85,
  };
  return fallback[job?.status] ?? 0.05;
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
    notInBlank: first.notInBlank != null
      ? Number(first.notInBlank)
      : rows.filter((row) => row.status === "not_in_blank").length,
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
