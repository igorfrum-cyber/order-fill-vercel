// Translation between the snake_case API contract and the camelCase shapes the
// UI works with. Keeping it here means no view module has to know the wire
// format, and a contract change is a one-file change.

export function mapJob(payload, absoluteUrl) {
  return {
    id: payload.id,
    type: payload.type,
    status: payload.status,
    brand: payload.brand || "",
    orderMonth: payload.order_month || "",
    error: payload.error || null,
    progress: Number.isFinite(Number(payload.progress)) ? Number(payload.progress) : null,
    progressMessage: payload.progress_message || "",
    outputFiles: (payload.output_files || []).map((file) => mapOutputFile(file, absoluteUrl)),
  };
}

export function mapOutputFile(file, absoluteUrl) {
  return {
    id: file.id,
    label: file.label || file.name || "Скачать файл",
    name: file.name,
    contentType: file.content_type || "",
    sizeBytes: file.size_bytes || 0,
    downloadUrl: absoluteUrl(file.download_path || ""),
  };
}

// The contract carries edit values as text, while the UI keeps whatever the
// report delivered: a number for untouched rows and a string once the reviewer
// types. Normalising here keeps that difference out of the request body.
export function toManualEditPayload(edit) {
  return {
    key: edit.key,
    value: edit.value == null ? "" : String(edit.value),
    comment: edit.comment == null ? "" : String(edit.comment),
  };
}

export function mapReport(payload) {
  return {
    jobId: payload.job_id || "",
    summary: mapSummary(payload.summary),
    rows: (payload.rows || []).map(mapReportRow),
  };
}

export function mapSummary(summary) {
  const source = summary || {};
  return {
    brand: source.brand || "",
    orderMonthLabel: source.order_month_label || "",
    adjustmentLabel: source.adjustment_label || "",
    actualMainPeriod: source.actual_main_period || "",
    actualPreviousPeriod: source.actual_previous_period || "",
    sourceCity: source.source_city || "",
    cityRule: source.city_rule || "",
    deliveryWeeks: source.delivery_weeks || "",
    filled: source.filled || 0,
    leftBlank: source.left_blank || 0,
    suspicious: source.suspicious || 0,
    unmatched: source.unmatched || 0,
    duplicates: source.duplicates || 0,
    notInBlank: source.not_in_blank || 0,
    blankDuplicateArticles: source.blank_duplicate_articles || 0,
  };
}

export function mapReportRow(row) {
  return {
    key: row.key,
    status: row.status,
    blankId: row.blank_id || "main",
    blankLabel: row.blank_label || "",
    blankRow: row.blank_row || "",
    blankQuantityCol: row.blank_quantity_col || null,
    blankArticle: row.blank_article || "",
    blankName: row.blank_name || "",
    blankUnit: row.blank_unit || "",
    blankBoxSize: row.blank_box_size || "",
    sourceRow: row.source_row ?? null,
    sourceArticle: row.source_article || "",
    sourceName: row.source_name || "",
    hasOrderedFact: row.has_ordered_fact || false,
    orderedFact: row.ordered_fact ?? null,
    sourceComment: row.source_comment || "",
    stock: row.stock ?? "",
    inTransit: row.in_transit ?? "",
    recommended: row.recommended ?? null,
    rounded: row.rounded ?? null,
    baseRounded: row.base_rounded ?? null,
    inserted: row.inserted ?? null,
    autoComment: row.auto_comment || "",
    adjustmentLabel: row.adjustment_label || "",
    boxAdjusted: row.box_adjusted || false,
    duplicate: row.duplicate || false,
    duplicateCandidates: (row.duplicate_candidates || []).map(mapDuplicateCandidate),
    editable: row.editable !== false,
    similarity: row.similarity || 0,
  };
}

function mapDuplicateCandidate(candidate) {
  return {
    sourceRow: candidate.source_row,
    sourceArticle: candidate.source_article || "",
    sourceName: candidate.source_name || "",
    recommended: candidate.recommended ?? null,
    rounded: candidate.rounded ?? null,
    stock: candidate.stock ?? "",
    inTransit: candidate.in_transit ?? "",
  };
}
