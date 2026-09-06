import { statusLabel } from "./reportModel.js";
import { rowCategory } from "./rowPresentation.js";

export function duplicateDescription(candidates = []) {
  if (!candidates.length) return "";
  return candidates
    .map((item) => `Строка ${item.sourceRow}: ${item.sourceName || ""}`)
    .join("; ");
}

export function isCleanupIssueRow(row) {
  const category = row.category || row.status;
  const reasons = row.matchReasons || {};
  return (
    category === "needs_decision" ||
    category === "not_in_source" ||
    category === "not_in_blank" ||
    reasons.volume === "conflict" ||
    reasons.form === "conflict" ||
    reasons.source === "name"
  );
}

export function issueReason(row) {
  const reasons = [];
  const match = row.matchReasons || {};
  const category = rowCategory(row);
  if (match.duplicates === "needs_choice") reasons.push("неоднозначный дубль");
  if (match.source === "chz" && match.duplicates === "needs_choice") reasons.push("неоднозначное объединение ЧЗ");
  if (match.source === "chz" && match.duplicates !== "needs_choice") reasons.push("ЧЗ без основной строки");
  if (match.volume === "conflict") reasons.push("конфликт объёма");
  if (match.form === "conflict") reasons.push("конфликт типа");
  if (match.source === "name") reasons.push("найдено только по названию");
  if (category === "not_in_source" || row.status === "not_in_source") {
    reasons.push("есть в бланке, но нет в 1С");
  }
  if (category === "not_in_blank" || row.status === "not_in_blank") {
    reasons.push("есть в 1С с потребностью, но нет в бланке");
  }
  if (row.status === "warning_name_only") reasons.push("В таблице заказа нет артикула, найдено только по названию");
  if (row.status === "warning_name_differs") reasons.push("Артикул найден, но название сильно отличается");
  if (row.status === "source_duplicate") reasons.push("В таблице заказа есть несколько строк с одним артикулом");
  if (row.duplicate && match.duplicates !== "needs_choice") reasons.push("Есть дублирующиеся кандидаты по артикулу");
  const duplicateText = duplicateDescription(row.duplicateCandidates || []);
  if (duplicateText) reasons.push(`Дубли в таблице: ${duplicateText}`);
  return [...new Set(reasons)].join("; ");
}

export function issueReportCsv(rows, getEdit = () => ({ comment: "" })) {
  const header = [
    "Статус",
    "Бланк",
    "Артикул в бланке",
    "Товар в бланке",
    "Объем",
    "Строка в таблице заказа",
    "Артикул в 1С",
    "Товар в 1С",
    "Проблема",
    "Комментарий менеджера",
  ];
  const body = rows.map((row) => {
    const edit = getEdit(row);
    return [
      statusLabel(row.status) || row.category || "",
      row.blankLabel,
      row.blankArticle,
      row.blankName,
      row.blankUnit,
      row.sourceRow || "",
      row.sourceArticle,
      row.sourceName,
      issueReason(row),
      edit.comment,
    ];
  });
  return [header, ...body].map((row) => row.map(csvCell).join(";")).join("\n");
}

function csvCell(value) {
  return `"${String(value ?? "").replaceAll('"', '""')}"`;
}
