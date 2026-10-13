

export function createOrderReport(context) {
  const {elements, state, actions} = context;
function statusLabel(status) {
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

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function combinedSummary(results) {
  const first = results[0]?.summary || {};
  return {
    ...first,
    filled: results.reduce((sum, result) => sum + result.summary.filled, 0),
    leftBlank: results.reduce((sum, result) => sum + result.summary.leftBlank, 0),
    suspicious: results.reduce((sum, result) => sum + result.summary.suspicious, 0),
    notInSource: results.reduce((sum, result) => sum + result.summary.unmatched, 0),
    notInBlank: state.currentReportRows.filter((row) => row.status === "not_in_blank").length,
    duplicates: state.currentReportRows.filter((row) => row.duplicate).length,
    blankDuplicateArticles: results.reduce((sum, result) => sum + (result.summary.blankDuplicateArticles || 0), 0),
    blankWarnings: results.flatMap((result) => result.summary.blankWarnings || []),
  };
}

function renderMetrics(summary) {
  const rows = [
    ["filled", "Заполнено", summary.filled],
    ["leftBlank", "Оставлено пустым", summary.leftBlank],
    ["suspicious", "Проверить", summary.suspicious],
    ["notInSource", "Нет в таблице", summary.notInSource],
    ["notInBlank", "Нет в бланке", summary.notInBlank],
    ["duplicates", "Дублей", summary.duplicates],
  ];
  elements.metricsEl.innerHTML = rows
    .map(([filter, label, value]) => `
      <button class="metric" type="button" data-filter="${filter}" aria-pressed="${state.activeFilter === filter ? "true" : "false"}">
        <strong>${value}</strong><span>${label}</span>
      </button>
    `)
    .join("");
  const cityNote = summary.cityRule
    ? ` ${summary.cityRule}: рекомендации пересчитаны, срок поставки ${summary.deliveryWeeks} нед.`
    : "";
  const blankNote = summary.blankDuplicateArticles
    ? ` Проверка бланка: найдено дублей артикулов ${summary.blankDuplicateArticles}.`
    : "";
  elements.periodNote.textContent = `${summary.brand}. Заказ на ${summary.orderMonthLabel}. Период: ${summary.actualMainPeriod}. Прошлый период: ${summary.actualPreviousPeriod}.${cityNote}${blankNote}`;
}

function initialComment(row) {
  return row.sourceComment || row.autoComment || "";
}

function rowEdit(row) {
  const key = row.key || `${row.blankId}:${row.blankRow}`;
  if (!state.editState.has(key)) {
    state.editState.set(key, { value: row.inserted ?? "", comment: initialComment(row) });
  }
  return state.editState.get(key);
}

function isIssueRow(row) {
  return row.status === "warning_name_differs" || row.status === "warning_name_only";
}

function isPriorityRow(row) {
  return isIssueRow(row) || Boolean(row.duplicate);
}

function rowMatchesFilter(row, filter) {
  if (!filter) return true;
  if (filter === "filled") return (row.status === "matched" || row.status === "matched_by_name") && row.inserted != null;
  if (filter === "leftBlank") return row.status === "left_blank_nonpositive";
  if (filter === "suspicious") return isIssueRow(row);
  if (filter === "notInSource") return row.status === "not_in_source";
  if (filter === "notInBlank") return row.status === "not_in_blank";
  if (filter === "duplicates") return Boolean(row.duplicate);
  return true;
}

function rowSearchText(row) {
  return [
    statusLabel(row.status),
    row.blankLabel,
    row.blankArticle,
    row.blankName,
    row.blankUnit,
    row.sourceArticle,
    row.sourceName,
    duplicateDescription(row),
  ].join(" ").toLowerCase();
}

function rowMatchesSearch(row, query) {
  const normalized = query.trim().toLowerCase();
  return !normalized || rowSearchText(row).includes(normalized);
}

function filterTitle(filter) {
  const labels = {
    filled: "Заполнено",
    leftBlank: "Оставлено пустым",
    suspicious: "Проверить",
    notInSource: "Нет в таблице",
    notInBlank: "Нет в бланке",
    duplicates: "Дубли",
  };
  return labels[filter] || "Все позиции";
}

function duplicateDescription(row) {
  const candidates = row.duplicateCandidates || [];
  if (!candidates.length) return "";
  return candidates
    .map((item) => `Строка ${item.sourceRow}: ${item.sourceName || ""}`)
    .join("; ");
}

function duplicateDetailsHtml(row) {
  const candidates = row.duplicateCandidates || [];
  if (!candidates.length) return "";
  const rows = candidates
    .map((item) => `
      <div class="duplicate-row">
        <span>Строка ${escapeHtml(item.sourceRow)}</span>
        <span>${escapeHtml(item.sourceName || "")}</span>
      </div>
    `)
    .join("");
  return `<div class="duplicate-details"><div>Дубли в таблице:</div>${rows}</div>`;
}

function baselineForReportRow(row) {
  if (Number(row.recommended) < 1.5 || Number(row.rounded) <= 0) return null;
  return Number(row.rounded);
}

function renderRows(targetBody, rows) {
  targetBody.innerHTML = rows
    .map((row) => {
      const cls = isPriorityRow(row) ? "warn" : row.status === "matched" || row.status === "matched_by_name" ? "ok" : "muted";
      const edit = rowEdit(row);
      const inserted = edit.value;
      const comment = edit.comment;
      const baseline = baselineForReportRow(row) ?? "";
      const rowKey = row.key || `${row.blankId}:${row.blankRow}`;
      const recommended = row.recommended == null ? "" : Number(row.recommended).toFixed(2);
      const match = row.status === "not_in_source" || row.status === "not_in_blank" ? "" : `${Math.round(Number(row.similarity || 0) * 100)}%`;
      const statusMeta = row.duplicate ? `<div class="row-meta">Дубль</div>` : "";
      const duplicateDetails = duplicateDetailsHtml(row);
      const orderCell = row.editable === false ? "" : `
        <input
          class="qty-input"
          type="number"
          min="0"
          step="1"
          inputmode="numeric"
          data-key="${escapeHtml(rowKey)}"
          data-blank-id="${escapeHtml(row.blankId)}"
          data-row="${row.blankRow}"
          data-initial-value="${escapeHtml(row.inserted ?? "")}"
          data-baseline-value="${baseline}"
          data-auto-comment="${escapeHtml(row.autoComment || "")}"
          value="${escapeHtml(inserted)}"
          aria-label="Количество для строки ${row.blankRow}"
        />
        <label class="budget-row-lock"><input type="checkbox" data-budget-lock="${escapeHtml(rowKey)}" ${state.budgetLocks.has(rowKey) ? 'checked' : ''}> Закрепить</label>
      `;
      const commentCell = row.editable === false ? "" : `
        <input
          class="comment-input"
          type="text"
          data-key="${escapeHtml(rowKey)}"
          data-blank-id="${escapeHtml(row.blankId)}"
          data-row="${row.blankRow}"
          value="${escapeHtml(comment)}"
          aria-label="Комментарий для строки ${row.blankRow}"
        />
      `;
      return `
        <tr>
          <td class="${cls}">${statusLabel(row.status)}${statusMeta}</td>
          <td>${escapeHtml(row.blankLabel)}</td>
          <td>${escapeHtml(row.blankArticle)}</td>
          <td>${escapeHtml(row.blankName)}${duplicateDetails}</td>
          <td>${escapeHtml(row.blankUnit)}</td>
          <td>${escapeHtml(row.stock ?? "")}</td>
          <td>${escapeHtml(row.inTransit ?? "")}</td>
          <td>${recommended}</td>
          <td>${row.hasOrderedFact ? escapeHtml(row.orderedFact) : ""}</td>
          <td>${escapeHtml(row.blankBoxSize ?? "")}</td>
          <td>${orderCell}</td>
          <td>${commentCell}</td>
          <td>${match}</td>
        </tr>
      `;
    })
    .join("");
  for (const row of targetBody.querySelectorAll("tr")) actions.updateCommentHint(row);
}

function renderReportView() {
  const query = elements.reportSearch.value;
  const hasSearch = query.trim() !== "";
  const visibleRows = state.currentReportRows.filter((row) => rowMatchesFilter(row, state.activeFilter) && rowMatchesSearch(row, query));

  elements.clearFilterButton.classList.toggle("hidden", !state.activeFilter && !hasSearch);

  if (state.activeFilter || hasSearch) {
    elements.prioritySection.classList.add("hidden");
    const title = hasSearch ? `Результаты поиска${state.activeFilter ? ` - ${filterTitle(state.activeFilter)}` : ""}` : `Только позиции - ${filterTitle(state.activeFilter)}`;
    elements.reportTitle.textContent = `${title}: ${visibleRows.length}`;
    renderRows(elements.reportBody, visibleRows);
    return;
  }

  const issueRows = state.currentReportRows.filter(isPriorityRow);
  const normalRows = state.currentReportRows.filter((row) => !isPriorityRow(row));
  elements.prioritySection.classList.toggle("hidden", issueRows.length === 0);
  renderRows(elements.priorityBody, issueRows);
  elements.reportTitle.textContent = `Все остальные позиции: ${normalRows.length}`;
  renderRows(elements.reportBody, normalRows);
}

function handleReportInput(event) {
  if (event.target.matches('[data-budget-lock]')) {
    event.target.checked ? state.budgetLocks.add(event.target.dataset.budgetLock) : state.budgetLocks.delete(event.target.dataset.budgetLock);
    return;
  }
  if (event.target.matches(".qty-input, .comment-input")) {
    const key = event.target.dataset.key;
    const current = state.editState.get(key) || { value: "", comment: "" };
    if (event.target.matches(".qty-input")) {
      current.value = event.target.value;
      if (String(current.value).trim() !== '' && Number(current.value) === 0) state.budgetLocks.add(key);
      event.target.closest('tr').querySelector('[data-budget-lock]').checked = state.budgetLocks.has(key);
    }
    if (event.target.matches(".comment-input")) current.comment = event.target.value;
    state.editState.set(key, current);
    const row = event.target.closest("tr");
    row?.classList.remove("invalid");
    if (row) actions.updateCommentHint(row);
  }
}
  return { statusLabel, escapeHtml, combinedSummary, renderMetrics, initialComment, rowEdit, isIssueRow, isPriorityRow, rowMatchesFilter, rowSearchText, rowMatchesSearch, filterTitle, duplicateDescription, duplicateDetailsHtml, baselineForReportRow, renderRows, renderReportView, handleReportInput };
}
