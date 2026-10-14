

export function createOrderIssues(context) {
  const {elements, state, actions} = context;
function missingInBlankRows(results) {
  const candidates = new Map();
  const matchedSourceRows = new Set();
  for (const result of results) {
    for (const item of result.sourceItemsForMissingBlank || []) {
      if (!candidates.has(item.sourceRow)) candidates.set(item.sourceRow, item);
    }
    for (const row of result.reportRows || []) {
      if (row.sourceRow) matchedSourceRows.add(row.sourceRow);
    }
  }
  return Array.from(candidates.values())
    .filter((item) => !matchedSourceRows.has(item.sourceRow))
    .map((item) => ({
      status: "not_in_blank",
      blankId: "source",
      blankLabel: "1С",
      adjustmentLabel: "",
      key: `source:${item.sourceRow}`,
      blankRow: item.sourceRow,
      blankQuantityCol: null,
      blankArticle: item.sourceArticle,
      blankName: item.sourceName,
      blankUnit: "",
      blankBoxSize: "",
      sourceRow: item.sourceRow,
      sourceArticle: item.sourceArticle,
      sourceName: item.sourceName,
      hasOrderedFact: item.hasOrderedFact,
      orderedFact: item.orderedFact,
      sourceComment: item.sourceComment,
      stock: item.stock,
      inTransit: item.inTransit,
      recommended: item.recommended,
      rounded: item.rounded,
      baseRounded: null,
      inserted: null,
      autoComment: "",
      boxAdjusted: false,
      duplicate: false,
      editable: false,
      similarity: 0,
    }));
}

function duplicateSignature(candidates) {
  return (candidates || [])
    .map((item) => Number(item.sourceRow))
    .filter((row) => Number.isInteger(row))
    .sort((left, right) => left - right)
    .join(":");
}

function sourceDuplicateRows(results) {
  const represented = new Set();
  for (const result of results) {
    for (const row of result.reportRows || []) {
      if (row.duplicate) represented.add(duplicateSignature(row.duplicateCandidates));
    }
  }

  const rows = [];
  const seen = new Set();
  for (const result of results) {
    for (const group of result.sourceDuplicateGroups || []) {
      const signature = duplicateSignature(group.candidates);
      if (!signature || seen.has(signature) || represented.has(signature)) continue;
      seen.add(signature);
      const first = group.candidates[0] || {};
      rows.push({
        status: "source_duplicate",
        blankId: "source",
        blankLabel: "1С",
        adjustmentLabel: "",
        key: `source-duplicate:${signature}`,
        blankRow: first.sourceRow || "",
        blankQuantityCol: null,
        blankArticle: first.sourceArticle || group.article,
        blankName: first.sourceName || "",
        blankUnit: "",
        blankBoxSize: "",
        sourceRow: first.sourceRow || null,
        sourceArticle: first.sourceArticle || group.article,
        sourceName: first.sourceName || "",
        hasOrderedFact: false,
        orderedFact: null,
        sourceComment: "",
        stock: first.stock ?? "",
        inTransit: first.inTransit ?? "",
        recommended: first.recommended ?? null,
        rounded: first.rounded ?? null,
        baseRounded: null,
        inserted: null,
        autoComment: "",
        boxAdjusted: false,
        duplicate: true,
        duplicateCandidates: group.candidates || [],
        editable: false,
        similarity: 0,
      });
    }
  }
  return rows;
}

function issueReportRows() {
  return state.currentReportRows.filter((row) => row.status === "warning_name_differs" || row.status === "warning_name_only" || row.status === "not_in_source" || row.duplicate);
}

function issueReason(row) {
  const reasons = [];
  if (row.status === "warning_name_only") reasons.push("В таблице заказа нет артикула, найдено только по названию");
  if (row.status === "warning_name_differs") reasons.push("Артикул найден, но название сильно отличается");
  if (row.status === "not_in_source") reasons.push("Позиция есть в бланке, но не найдена в таблице заказа");
  if (row.status === "source_duplicate") reasons.push("В таблице заказа есть несколько строк с одним артикулом");
  if (row.duplicate) reasons.push("Есть дублирующиеся кандидаты по артикулу");
  const duplicateText = actions.duplicateDescription(row);
  if (duplicateText) reasons.push(`Дубли в таблице: ${duplicateText}`);
  return reasons.join("; ");
}

function csvCell(value) {
  return `"${String(value ?? "").replaceAll('"', '""')}"`;
}

function issueReportCsv() {
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
  const rows = issueReportRows().map((row) => {
    const edit = actions.rowEdit(row);
    return [
      actions.statusLabel(row.status),
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
  return [header, ...rows].map((row) => row.map(csvCell).join(";")).join("\n");
}
  return { missingInBlankRows, duplicateSignature, sourceDuplicateRows, issueReportRows, issueReason, csvCell, issueReportCsv };
}
