import { rowMatchesFilter } from "./reportModel.js";

export const REPORT_TABS = [
  { key: "needs_decision", label: "Требует решения" },
  { key: "not_in_source", label: "Нет в 1С" },
  { key: "check_name_or_volume", label: "Проверить название или объём" },
  { key: "not_in_blank", label: "Нет в бланке" },
  { key: "to_order", label: "К заказу" },
  { key: "order_not_needed", label: "Заказ не нужен" },
  { key: "all", label: "Все" },
];

export const FILL_TABS = REPORT_TABS;

export const FILL_COMPOSITION_ORDER = ["to_order", "order_not_needed", "check_name_or_volume", "needs_decision"];

export const MATCH_LAYER_TABS = ["not_in_source", "not_in_blank"];

const MATCH_LAYER_HINTS = {
  not_in_source: "Эти позиции бланка не нашлись в 1С. Если объединение не сработало — сверьте артикул и название или выгрузите отчёт для 1С.",
  not_in_blank: "Эти позиции 1С не нашлись в бланке. Их не будет в файле поставщика, пока не появятся в бланке.",
};

const FILTER_BY_TAB = {
  to_order: "filled",
  order_not_needed: "leftBlank",
  check_name_or_volume: "suspicious",
  needs_decision: "duplicates",
  not_in_source: "notInSource",
  not_in_blank: "notInBlank",
};

export function rowCategory(row) {
  if (row.category) return row.category;
  if (row.duplicate || row.status === "source_duplicate" || row.status === "warning_name_only") return "needs_decision";
  if (row.status === "not_in_source") return "not_in_source";
  if (row.status === "warning_name_differs") return "check_name_or_volume";
  if (row.status === "not_in_blank") return "not_in_blank";
  if (row.status === "left_blank_nonpositive") return "order_not_needed";
  return row.inserted == null ? "order_not_needed" : "to_order";
}

export function presentationStatus(row) {
  return rowCategory(row);
}

export function rowMatchesTab(row, tab) {
  if (!tab || tab === "all") return true;
  if (row.category) return rowCategory(row) === tab;
  const filter = FILTER_BY_TAB[tab];
  if (filter) return rowMatchesFilter(row, filter) || rowCategory(row) === tab;
  return rowCategory(row) === tab;
}

export function countByTab(rows) {
  const counts = {
    all: rows.length,
    needs_decision: 0,
    not_in_source: 0,
    check_name_or_volume: 0,
    not_in_blank: 0,
    to_order: 0,
    order_not_needed: 0,
  };
  for (const row of rows) {
    const status = rowCategory(row);
    counts[status] = (counts[status] || 0) + 1;
  }
  return counts;
}

export function rowMatchesQuery(row, query) {
  const normalized = String(query || "").trim().toLowerCase();
  if (!normalized) return true;
  return [
    row.blankArticle,
    row.blankName,
    row.sourceArticle,
    row.sourceName,
  ].some((value) => String(value || "").toLowerCase().includes(normalized));
}

export function visibleReportRows(rows, { tab = "all", query = "" } = {}) {
  return rows.filter((row) => rowMatchesTab(row, tab) && rowMatchesQuery(row, query));
}

export function displayArticle(row) {
  return String(row.blankArticle || "").trim() || String(row.sourceArticle || "").trim();
}

export function displayName(row) {
  return String(row.blankName || "").trim() || String(row.sourceName || "").trim();
}

export function matchPercent(row) {
  if (rowCategory(row) === "not_in_source" || rowCategory(row) === "not_in_blank") return null;
  return Math.round(Number(row.similarity || 0) * 100);
}

export function matchReasonLabel(row) {
  const reasons = row.matchReasons || {};
  if (reasons.duplicates === "needs_choice") return "Дубль: нужно выбрать";
  if (reasons.duplicates === "chosen_best") return "Дубль: выбран лучший";
  if (reasons.volume === "conflict") return "Проверить объём";
  if (reasons.form === "conflict") return "Проверить тип";
  if (reasons.source === "name") return "Найдено только по названию";
  if (reasons.source === "none") return "Нет пары";
  if (reasons.article === "exact" || reasons.article === "alias") return "Надёжно";
  return "";
}

export function boxStep(row) {
  const size = Number(row.blankBoxSize);
  return Number.isFinite(size) && size > 0 ? size : 1;
}

export function pairedRowCount(counts) {
  return FILL_COMPOSITION_ORDER.reduce((sum, key) => sum + (counts[key] || 0), 0);
}

export function fillReadiness(counts) {
  const paired = pairedRowCount(counts);
  return paired ? (counts.to_order || 0) / paired : 0;
}

export function firstReviewTab(counts = {}) {
  if (counts.needs_decision) return "needs_decision";
  if (counts.not_in_source) return "not_in_source";
  if (counts.check_name_or_volume) return "check_name_or_volume";
  if (counts.not_in_blank) return "not_in_blank";
  if (counts.to_order) return "to_order";
  return "order_not_needed";
}

export function reviewQueueLine(counts = {}) {
  const n = counts.needs_decision || 0;
  return n ? `Осталось ${n} спорных` : "";
}

export function visibleFillTabs() {
  return REPORT_TABS;
}

export function matchLayerHint(tab) {
  return MATCH_LAYER_HINTS[tab] || "";
}

export function reviewTableHeaders() {
  return [
    { key: "bar", label: "", align: "left" },
    { key: "article", label: "Артикул", align: "left" },
    { key: "name", label: "Товар", align: "left" },
    { key: "unit", label: "Объём", align: "right" },
    { key: "stock", label: "Остаток", align: "right" },
    { key: "transit", label: "В пути", align: "right" },
    { key: "recommended", label: "Расчёт", align: "right" },
    { key: "inserted", label: "Вставлено", align: "right" },
    { key: "match", label: "Причина", align: "right" },
    { key: "comment", label: "Комментарий", align: "left" },
  ];
}

export function attentionReason(row) {
  const label = matchReasonLabel(row);
  if (label) return label;
  if (row.status === "warning_name_differs") {
    return "Название отличается от таблицы заказа.";
  }
  if (row.status === "not_in_source") {
    return "Позиция есть в бланке, но не нашлась в 1С.";
  }
  return "";
}

export const reviewCommentBanner =
  "Есть строки, где изменено значение «Вставлено», но не заполнен новый комментарий.";

export const matchingDecisionBanner =
  "Есть строки, которые требуют решения, прежде чем скачивать файлы.";

export const commentGateTitle = "Сначала напишите, почему изменили количество";
export const commentGateHint = "Эти строки не пускаем в файлы, пока не будет комментария.";
export const commentGateConfirm = "Продолжить";
export const commentGateCommentPlaceholder = "Почему изменили количество";

export function canProceedPastDuplicates({ duplicateKeys = [], acknowledgedKeys = new Set() } = {}) {
  return duplicateKeys.every((key) => acknowledgedKeys.has(key));
}

export function needsAcknowledgement(row) {
  return presentationStatus(row) === "needs_decision";
}

export function isDuplicateDecision(row) {
  return Boolean(
    row.duplicate || row.matchReasons?.duplicates === "needs_choice" || row.status === "source_duplicate",
  );
}

export function decisionHint(rows = []) {
  const pending = rows.filter(needsAcknowledgement);
  if (!pending.length) return "";
  if (pending.every(isDuplicateDecision)) {
    return "В таблице заказа несколько строк на одну позицию бланка. На каждой строке отметьте «оставляю», когда разобрали конфликт.";
  }
  if (pending.some(isDuplicateDecision)) {
    return "Есть дубли и сомнительные пары. На каждой строке отметьте «оставляю», когда разобрали.";
  }
  return "Эти позиции найдены только по названию. На каждой строке отметьте «оставляю», если пара верная.";
}

export function quantityDisplay(value) {
  if (value == null || value === "") return "";
  const number = Number(value);
  if (!Number.isFinite(number)) return String(value);
  return Number.isInteger(number) ? String(number) : String(number);
}
