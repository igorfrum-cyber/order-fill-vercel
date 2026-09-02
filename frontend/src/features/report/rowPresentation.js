import { isIssueRow, rowMatchesFilter } from "./reportModel.js";

export const FILL_TABS = [
  { key: "empty", label: "Пусто" },
  { key: "check", label: "Нужно проверить" },
  { key: "duplicate", label: "Дубли" },
  { key: "not_in_table", label: "Нет в таблице" },
  { key: "not_in_blank", label: "Нет в бланке" },
  { key: "filled", label: "Заполнено" },
  { key: "all", label: "Все" },
];

export const FILL_COMPOSITION_ORDER = ["filled", "empty", "check", "duplicate"];

export const MATCH_LAYER_TABS = ["not_in_table", "not_in_blank"];

const MATCH_LAYER_HINTS = {
  not_in_table: "Эти позиции бланка не нашлись в таблице заказа. Если объединение не сработало — сверьте артикул и название или выгрузите отчёт для 1С.",
  not_in_blank: "Эти позиции таблицы заказа не нашлись в бланке. Их не будет в файле поставщика, пока не появятся в бланке.",
};

const FILTER_BY_TAB = {
  filled: "filled",
  empty: "leftBlank",
  check: "suspicious",
  duplicate: "duplicates",
  not_in_table: "notInSource",
  not_in_blank: "notInBlank",
};

export function presentationStatus(row) {
  if (row.duplicate || row.status === "source_duplicate") return "duplicate";
  if (row.status === "not_in_source") return "not_in_table";
  if (row.status === "not_in_blank") return "not_in_blank";
  if (isIssueRow(row)) return "check";
  if (row.status === "left_blank_nonpositive") return "empty";
  if ((row.status === "matched" || row.status === "matched_by_name") && row.inserted != null) return "filled";
  return "empty";
}

export function rowMatchesTab(row, tab) {
  if (!tab || tab === "all") return true;
  return rowMatchesFilter(row, FILTER_BY_TAB[tab]);
}

export function countByTab(rows) {
  const counts = { all: rows.length, filled: 0, empty: 0, check: 0, duplicate: 0, not_in_table: 0, not_in_blank: 0 };
  for (const row of rows) {
    const status = presentationStatus(row);
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
  if (row.status === "not_in_source" || row.status === "not_in_blank") return null;
  return Math.round(Number(row.similarity || 0) * 100);
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
  return paired ? (counts.filled || 0) / paired : 0;
}

export function visibleFillTabs(counts) {
  return FILL_TABS.filter((tab) => {
    if (tab.key === "all" || MATCH_LAYER_TABS.includes(tab.key)) return true;
    return (counts[tab.key] || 0) > 0;
  });
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
    { key: "match", label: "Похоже", align: "right" },
    { key: "comment", label: "Комментарий", align: "left" },
  ];
}

export function attentionReason(row) {
  if (row.status === "warning_name_differs") {
    return "Название отличается от таблицы заказа.";
  }
  if (row.status === "not_in_source") {
    return "Позиция есть в бланке, но не нашлась в таблице заказа.";
  }
  return "";
}

export const reviewCommentBanner =
  "Есть строки, где изменено значение «Вставлено», но не заполнен новый комментарий.";

export function canProceedPastDuplicates({ duplicateKeys = [], acknowledgedKeys = new Set() } = {}) {
  return duplicateKeys.every((key) => acknowledgedKeys.has(key));
}

export function quantityDisplay(value) {
  if (value == null || value === "") return "";
  const number = Number(value);
  if (!Number.isFinite(number)) return String(value);
  return Number.isInteger(number) ? String(number) : String(number);
}
