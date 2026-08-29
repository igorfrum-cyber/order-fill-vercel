import { statusLabel } from "./reportModel.js";

export function duplicateDescription(candidates = []) {
  if (!candidates.length) return "";
  return candidates
    .map((item) => `Строка ${item.sourceRow}: ${item.sourceName || ""}`)
    .join("; ");
}

export function issueReason(row) {
  const reasons = [];
  if (row.status === "warning_name_only") reasons.push("В таблице заказа нет артикула, найдено только по названию");
  if (row.status === "warning_name_differs") reasons.push("Артикул найден, но название сильно отличается");
  if (row.status === "not_in_source") reasons.push("Позиция есть в бланке, но не найдена в таблице заказа");
  if (row.status === "source_duplicate") reasons.push("В таблице заказа есть несколько строк с одним артикулом");
  if (row.duplicate) reasons.push("Есть дублирующиеся кандидаты по артикулу");
  const duplicateText = duplicateDescription(row.duplicateCandidates || []);
  if (duplicateText) reasons.push(`Дубли в таблице: ${duplicateText}`);
  return reasons.join("; ");
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
      statusLabel(row.status),
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
