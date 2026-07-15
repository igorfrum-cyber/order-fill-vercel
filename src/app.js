import "./styles.css";
import {
  applyFinalEdits,
  fillWorkbook,
  loadXlsx,
  normalizeOrderValue,
  outputFileName,
  saveXlsx,
  sourceOutputFileName,
} from "./workbookProcessor.js";

const form = document.querySelector("#uploadForm");
const statusEl = document.querySelector("#status");
const brandSelect = document.querySelector("#brandSelect");
const orderMonth = document.querySelector("#orderMonth");
const sourceFile = document.querySelector("#sourceFile");
const blankFile = document.querySelector("#blankFile");
const homeFile = document.querySelector("#homeFile");
const proffFile = document.querySelector("#proffFile");
const sourceName = document.querySelector("#sourceName");
const blankName = document.querySelector("#blankName");
const homeName = document.querySelector("#homeName");
const proffName = document.querySelector("#proffName");
const blankField = document.querySelector("#blankField");
const homeField = document.querySelector("#homeField");
const proffField = document.querySelector("#proffField");
const resultEl = document.querySelector("#result");
const metricsEl = document.querySelector("#metrics");
const periodNote = document.querySelector("#periodNote");
const reportBody = document.querySelector("#reportBody");
const priorityBody = document.querySelector("#priorityBody");
const prioritySection = document.querySelector("#prioritySection");
const reportTitle = document.querySelector("#reportTitle");
const reportSearch = document.querySelector("#reportSearch");
const clearFilterButton = document.querySelector("#clearFilterButton");
const adjustmentHeader = document.querySelector("#adjustmentHeader");
const priorityAdjustmentHeader = document.querySelector("#priorityAdjustmentHeader");
const issueReportButton = document.querySelector("#issueReportButton");
const downloadButton = document.querySelector("#downloadButton");
const downloadLinks = document.querySelector("#downloadLinks");
const submitButton = form.querySelector("button");

let currentResults = [];
let currentReportRows = [];
let currentBlankWorkbooks = new Map();
let currentSourceWorkbook = null;
let currentBlankOutputNames = new Map();
let currentSourceOutputName = "order заполненная таблица.xlsx";
let currentDownloadUrls = [];
let isFormFilled = false;
let activeFilter = null;
let editState = new Map();

function setDefaultOrderMonth() {
  const date = new Date();
  date.setMonth(date.getMonth() + 1);
  orderMonth.value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

setDefaultOrderMonth();

function bindFileName(input, output) {
  input.addEventListener("change", () => {
    output.textContent = input.files[0]?.name || ".xlsx, .xlsm или .xls";
    resetFillState();
  });
}

bindFileName(sourceFile, sourceName);
bindFileName(blankFile, blankName);
bindFileName(homeFile, homeName);
bindFileName(proffFile, proffName);

function selectedBrand() {
  return brandSelect.value || "angiopharm";
}

function adjustmentLabelForBrand(brand) {
  if (brand === "christina") return "Кратность";
  if (brand === "levissime") return "Кол-во в уп.";
  if (brand === "sothys") return "Округление";
  return "Шт. в коробке";
}

function mainBlankLabelForBrand(brand) {
  if (brand === "levissime") return "LeviSsime";
  if (brand === "sothys") return "SOTHYS";
  return "ANGIO";
}

function configureBrandFields() {
  const brand = selectedBrand();
  const isChristina = brand === "christina";
  blankField.classList.toggle("hidden", isChristina);
  homeField.classList.toggle("hidden", !isChristina);
  proffField.classList.toggle("hidden", !isChristina);
  blankFile.required = !isChristina;
  homeFile.required = isChristina;
  proffFile.required = isChristina;
  adjustmentHeader.textContent = adjustmentLabelForBrand(brand);
  priorityAdjustmentHeader.textContent = adjustmentLabelForBrand(brand);
  resetFillState();
}

brandSelect.addEventListener("change", configureBrandFields);
orderMonth.addEventListener("change", resetFillState);
configureBrandFields();

function setSubmitButtonState(state) {
  submitButton.classList.toggle("completed", state === "completed");

  if (state === "processing") {
    submitButton.disabled = true;
    submitButton.innerHTML = "<span>✓</span> Заполняю...";
    return;
  }

  if (state === "completed") {
    submitButton.disabled = true;
    submitButton.innerHTML = "<span>✓</span> Бланк заполнен";
    return;
  }

  submitButton.disabled = false;
  submitButton.innerHTML = "<span>✓</span> Заполнить бланк";
}

function resetFillState() {
  if (!isFormFilled && !currentResults.length && resultEl.classList.contains("hidden")) {
    setSubmitButtonState("ready");
    return;
  }

  isFormFilled = false;
  currentResults = [];
  currentReportRows = [];
  currentBlankWorkbooks = new Map();
  currentBlankOutputNames = new Map();
  currentSourceWorkbook = null;
  activeFilter = null;
  editState = new Map();
  reportSearch.value = "";
  resultEl.classList.add("hidden");
  downloadButton.disabled = true;
  issueReportButton.disabled = true;
  clearDownloadLinks();
  statusEl.textContent = "Готов к загрузке";
  setSubmitButtonState("ready");
}

function statusLabel(status) {
  const labels = {
    matched: "Заполнено",
    matched_by_name: "По названию",
    warning_name_differs: "Проверить название",
    warning_name_only: "Проверить без артикула",
    left_blank_nonpositive: "Пусто",
    unmatched: "Не найдено",
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
    unmatched: results.reduce((sum, result) => sum + result.summary.unmatched, 0),
    duplicates: results.reduce((sum, result) => sum + result.summary.duplicates, 0),
    blankDuplicateArticles: results.reduce((sum, result) => sum + (result.summary.blankDuplicateArticles || 0), 0),
    blankWarnings: results.flatMap((result) => result.summary.blankWarnings || []),
  };
}

function renderMetrics(summary) {
  const rows = [
    ["filled", "Заполнено", summary.filled],
    ["leftBlank", "Оставлено пустым", summary.leftBlank],
    ["suspicious", "Проверить", summary.suspicious],
    ["unmatched", "Не найдено", summary.unmatched],
    ["duplicates", "Дублей", summary.duplicates],
  ];
  metricsEl.innerHTML = rows
    .map(([filter, label, value]) => `
      <button class="metric" type="button" data-filter="${filter}" aria-pressed="${activeFilter === filter ? "true" : "false"}">
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
  periodNote.textContent = `${summary.brand}. Заказ на ${summary.orderMonthLabel}. Период: ${summary.actualMainPeriod}. Прошлый период: ${summary.actualPreviousPeriod}.${cityNote}${blankNote}`;
}

function initialComment(row) {
  return row.sourceComment || row.autoComment || "";
}

function rowEdit(row) {
  const key = row.key || `${row.blankId}:${row.blankRow}`;
  if (!editState.has(key)) {
    editState.set(key, { value: row.inserted ?? "", comment: initialComment(row) });
  }
  return editState.get(key);
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
  if (filter === "unmatched") return row.status === "unmatched";
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
    unmatched: "Не найдено",
    duplicates: "Дубли",
  };
  return labels[filter] || "Все позиции";
}

function suggestionHtml(suggestions) {
  if (!suggestions.length) return "";
  const items = suggestions
    .map((item) => {
      const article = item.article ? `${escapeHtml(item.article)} - ` : "";
      const score = Math.round(Number(item.score || 0) * 100);
      return `<li>${article}${escapeHtml(item.name)} <span>${score}%</span></li>`;
    })
    .join("");
  return `<div class="suggestions"><strong>Похожие:</strong><ul>${items}</ul></div>`;
}

function renderRows(targetBody, rows) {
  targetBody.innerHTML = rows
    .map((row) => {
      const cls = isPriorityRow(row) ? "warn" : row.status === "matched" || row.status === "matched_by_name" ? "ok" : "muted";
      const edit = rowEdit(row);
      const inserted = edit.value;
      const comment = edit.comment;
      const baseline = Number(row.recommended) < 1.5 || Number(row.rounded) <= 0 ? "" : row.rounded;
      const rowKey = row.key || `${row.blankId}:${row.blankRow}`;
      const recommended = row.recommended == null ? "" : Number(row.recommended).toFixed(2);
      const match = row.status === "unmatched" ? "" : `${Math.round(Number(row.similarity || 0) * 100)}%`;
      const suggestions = suggestionHtml(row.suggestions || []);
      const statusMeta = row.duplicate ? `<div class="row-meta">Дубль</div>` : "";
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
          <td>${escapeHtml(row.blankName)}${suggestions}</td>
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
  for (const row of targetBody.querySelectorAll("tr")) updateCommentHint(row);
}

function renderReportView() {
  const query = reportSearch.value;
  const hasSearch = query.trim() !== "";
  const visibleRows = currentReportRows.filter((row) => rowMatchesFilter(row, activeFilter) && rowMatchesSearch(row, query));

  clearFilterButton.classList.toggle("hidden", !activeFilter && !hasSearch);

  if (activeFilter || hasSearch) {
    prioritySection.classList.add("hidden");
    const title = hasSearch ? `Результаты поиска${activeFilter ? ` - ${filterTitle(activeFilter)}` : ""}` : `Только позиции - ${filterTitle(activeFilter)}`;
    reportTitle.textContent = `${title}: ${visibleRows.length}`;
    renderRows(reportBody, visibleRows);
    return;
  }

  const issueRows = currentReportRows.filter(isPriorityRow);
  const normalRows = currentReportRows.filter((row) => !isPriorityRow(row));
  prioritySection.classList.toggle("hidden", issueRows.length === 0);
  renderRows(priorityBody, issueRows);
  reportTitle.textContent = `Все остальные позиции: ${normalRows.length}`;
  renderRows(reportBody, normalRows);
}

async function loadWorkbook(file) {
  const buffer = await file.arrayBuffer();
  return loadXlsx(await normalizeWorkbookBytes(buffer, file.name));
}

function isLegacyXls(fileName) {
  return /\.xls$/i.test(fileName) && !/\.xlsx$/i.test(fileName) && !/\.xlsm$/i.test(fileName);
}

async function normalizeWorkbookBytes(buffer, fileName) {
  if (!isLegacyXls(fileName)) return buffer;
  try {
    const { read: readSpreadsheet, write: writeSpreadsheet } = await import("xlsx");
    const workbook = readSpreadsheet(buffer, {
      type: "array",
      cellFormula: true,
      cellStyles: true,
      cellDates: false,
    });
    return writeSpreadsheet(workbook, {
      bookType: "xlsx",
      type: "array",
    });
  } catch {
    throw new Error(`Файл «${fileName}» в старом формате .xls не удалось автоматически прочитать. Откройте его в Excel и сохраните как .xlsx.`);
  }
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const brand = selectedBrand();
  const blankInputs = brand === "christina"
    ? [
        { id: "home", label: "HOME", file: homeFile.files[0] },
        { id: "proff", label: "PROFF", file: proffFile.files[0] },
      ]
    : [{ id: "main", label: mainBlankLabelForBrand(brand), file: blankFile.files[0] }];
  if (!sourceFile.files[0] || blankInputs.some((item) => !item.file)) return;

  statusEl.textContent = "Обработка...";
  setSubmitButtonState("processing");
  downloadButton.disabled = true;
  issueReportButton.disabled = true;
  resultEl.classList.add("hidden");
  clearDownloadLinks();
  isFormFilled = false;
  currentResults = [];
  currentReportRows = [];
  activeFilter = null;
  editState = new Map();
  currentBlankWorkbooks = new Map();
  currentBlankOutputNames = new Map();
  currentSourceWorkbook = null;

  try {
    const sourceWorkbook = await loadWorkbook(sourceFile.files[0]);
    const blankWorkbooks = await Promise.all(blankInputs.map((item) => loadWorkbook(item.file)));
    const results = blankInputs.map((item, index) => fillWorkbook({
      sourceWorkbook,
      blankWorkbook: blankWorkbooks[index],
      orderMonth: orderMonth.value,
      brand,
      blankId: item.id,
      blankLabel: item.label,
    }));

    currentResults = results;
    currentSourceWorkbook = sourceWorkbook;
    currentBlankWorkbooks = new Map(results.map((result) => [result.blankId, result.blankWorkbook]));
    for (const [index, item] of blankInputs.entries()) {
      currentBlankWorkbooks.set(item.id, results[index].blankWorkbook);
      currentBlankOutputNames.set(item.id, outputFileName(item.file.name));
    }
    currentSourceOutputName = sourceOutputFileName(sourceFile.files[0].name);

    const rows = results.flatMap((result) => result.reportRows);
    currentReportRows = rows;
    editState = new Map(rows.map((row) => [row.key || `${row.blankId}:${row.blankRow}`, { value: row.inserted ?? "", comment: initialComment(row) }]));
    renderMetrics(combinedSummary(results));
    renderReportView();
    resultEl.classList.remove("hidden");
    downloadButton.disabled = false;
    issueReportButton.disabled = !issueReportRows().length;
    statusEl.textContent = "Готово";
    isFormFilled = true;
    setSubmitButtonState("completed");
  } catch (error) {
    statusEl.textContent = "Ошибка";
    setSubmitButtonState("ready");
    alert(error.message || "Не удалось обработать файлы.");
  } finally {
    if (!isFormFilled) setSubmitButtonState("ready");
  }
});

function collectEdits() {
  return currentReportRows
    .filter((row) => row.editable !== false)
    .map((row) => {
      const key = row.key || `${row.blankId}:${row.blankRow}`;
      const edit = rowEdit(row);
      return {
        key,
        blankId: row.blankId,
        blankRow: Number(row.blankRow),
        value: edit.value,
        comment: edit.comment,
      };
    });
}

function issueReportRows() {
  return currentReportRows.filter((row) => row.status === "warning_name_differs" || row.status === "warning_name_only" || row.status === "unmatched" || row.duplicate);
}

function issueReason(row) {
  const reasons = [];
  if (row.status === "warning_name_only") reasons.push("В таблице заказа нет артикула, найдено только по названию");
  if (row.status === "warning_name_differs") reasons.push("Артикул найден, но название сильно отличается");
  if (row.status === "unmatched") reasons.push("Артикул из бланка не найден в таблице заказа");
  if (row.duplicate) reasons.push("Есть дублирующиеся кандидаты по артикулу");
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
    "Похожие варианты",
    "Комментарий менеджера",
  ];
  const rows = issueReportRows().map((row) => {
    const edit = rowEdit(row);
    const suggestions = (row.suggestions || [])
      .map((item) => `${item.article ? `${item.article} - ` : ""}${item.name} (${Math.round(Number(item.score || 0) * 100)}%)`)
      .join(" | ");
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
      suggestions,
      edit.comment,
    ];
  });
  return [header, ...rows].map((row) => row.map(csvCell).join(";")).join("\n");
}

function clearDownloadLinks() {
  for (const url of currentDownloadUrls) URL.revokeObjectURL(url);
  currentDownloadUrls = [];
  downloadLinks.classList.add("hidden");
  downloadLinks.innerHTML = "";
}

function validateEdits() {
  let invalidCount = 0;
  for (const row of reportBody.querySelectorAll("tr")) row.classList.remove("invalid");
  for (const row of priorityBody.querySelectorAll("tr")) row.classList.remove("invalid");

  for (const rowInfo of currentReportRows) {
    if (rowInfo.editable === false) continue;
    const key = rowInfo.key || `${rowInfo.blankId}:${rowInfo.blankRow}`;
    const edit = rowEdit(rowInfo);
    const initial = rowInfo.inserted == null ? null : Number(rowInfo.inserted);
    const baseline = Number(rowInfo.recommended) < 1.5 || Number(rowInfo.rounded) <= 0 ? null : Number(rowInfo.rounded);
    const autoComment = (rowInfo.autoComment || "").trim().toLowerCase();
    let value;
    try {
      value = normalizeOrderValue(edit.value);
    } catch {
      document.querySelectorAll(`[data-key="${CSS.escape(key)}"]`).forEach((input) => input.closest("tr")?.classList.add("invalid"));
      invalidCount += 1;
      continue;
    }
    const comment = edit.comment.trim();
    const requiresComment = value !== baseline;
    const stillAutoComment = autoComment && comment.toLowerCase() === autoComment;
    const autoCommentAllowed = stillAutoComment && value === initial;
    if (requiresComment && (!comment || (stillAutoComment && !autoCommentAllowed))) {
      document.querySelectorAll(`[data-key="${CSS.escape(key)}"]`).forEach((input) => input.closest("tr")?.classList.add("invalid"));
      invalidCount += 1;
    }
  }

  if (invalidCount > 0) {
    const firstInvalid = priorityBody.querySelector("tr.invalid") || reportBody.querySelector("tr.invalid");
    firstInvalid?.scrollIntoView({ block: "center", behavior: "smooth" });
    alert("Есть строки, где изменено значение «Вставлено», но не заполнен новый комментарий.");
    return false;
  }
  return true;
}

function isManualDeviation(rowInfo) {
  if (rowInfo.editable === false) return false;
  const edit = rowEdit(rowInfo);
  let value;
  try {
    value = normalizeOrderValue(edit.value);
  } catch {
    return true;
  }
  const baseline = Number(rowInfo.recommended) < 1.5 || Number(rowInfo.rounded) <= 0 ? null : Number(rowInfo.rounded);
  return value !== baseline;
}

function confirmQualityWarnings() {
  const issueCount = currentReportRows.filter((row) => row.status === "warning_name_differs" || row.status === "warning_name_only").length;
  const duplicateCount = currentReportRows.filter((row) => row.duplicate).length;
  const unmatchedCount = currentReportRows.filter((row) => row.status === "unmatched").length;
  const manualCount = currentReportRows.filter(isManualDeviation).length;
  const blankDuplicateCount = currentResults.reduce((sum, result) => sum + (result.summary.blankDuplicateArticles || 0), 0);
  const total = issueCount + duplicateCount + unmatchedCount + manualCount + blankDuplicateCount;
  if (!total) return true;

  const lines = [
    `Проверьте ${total} спорных строк/ситуаций перед скачиванием.`,
    issueCount ? `Проверить: ${issueCount}` : "",
    duplicateCount ? `Дубли: ${duplicateCount}` : "",
    unmatchedCount ? `Не найдено: ${unmatchedCount}` : "",
    manualCount ? `Ручные отклонения: ${manualCount}` : "",
    blankDuplicateCount ? `Дубли артикулов в бланке: ${blankDuplicateCount}` : "",
    "",
    "Продолжить скачивание?",
  ].filter((line) => line !== "").join("\n");
  return window.confirm(lines);
}

function rowNeedsComment(row) {
  const qtyInput = row.querySelector(".qty-input");
  const commentInput = row.querySelector(".comment-input");
  if (!qtyInput || !commentInput) return false;

  const initial = qtyInput.dataset.initialValue === "" ? null : Number(qtyInput.dataset.initialValue);
  const baseline = qtyInput.dataset.baselineValue === "" ? null : Number(qtyInput.dataset.baselineValue);
  const autoComment = (qtyInput.dataset.autoComment || "").trim().toLowerCase();
  let value;
  try {
    value = normalizeOrderValue(qtyInput.value);
  } catch {
    return false;
  }

  const comment = commentInput.value.trim();
  const stillAutoComment = autoComment && comment.toLowerCase() === autoComment;
  const autoCommentAllowed = stillAutoComment && value === initial;
  return value !== baseline && (!comment || (stillAutoComment && !autoCommentAllowed));
}

function updateCommentHint(row) {
  const commentInput = row.querySelector(".comment-input");
  if (!commentInput) return;
  commentInput.classList.toggle("needs-comment", rowNeedsComment(row));
}

function triggerDownload(url, fileName) {
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
}

function downloadIssueReport() {
  const rows = issueReportRows();
  if (!rows.length) {
    alert("Нет спорных строк для отчета.");
    return;
  }
  const blob = new Blob([`\ufeff${issueReportCsv()}`], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  triggerDownload(url, "отчет для исправления в 1С.csv");
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function prepareDownloadLinks(files) {
  clearDownloadLinks();
  currentDownloadUrls = files.map((file) => URL.createObjectURL(file.blob));

  downloadLinks.innerHTML = files
    .map((file, index) => `<a class="file-link" href="${currentDownloadUrls[index]}" download="${escapeHtml(file.name)}">${escapeHtml(file.label)}</a>`)
    .join("");
  downloadLinks.classList.remove("hidden");

  currentDownloadUrls.forEach((url, index) => {
    window.setTimeout(() => triggerDownload(url, files[index].name), index * 250);
  });
}

downloadButton.addEventListener("click", async () => {
  if (!currentResults.length || !currentBlankWorkbooks.size || !currentSourceWorkbook) {
    alert("Сначала заполните бланк.");
    return;
  }
  if (!validateEdits()) return;
  if (!confirmQualityWarnings()) return;

  downloadButton.disabled = true;
  statusEl.textContent = "Сохраняю правки...";
  try {
    const edits = collectEdits();
    const files = [];
    for (const result of currentResults) {
      const edited = applyFinalEdits({
        blankWorkbook: result.blankWorkbook,
        sourceWorkbook: currentSourceWorkbook,
        reportRows: result.reportRows,
        edits,
        brand: selectedBrand(),
      });
      result.blankWorkbook = edited.blankWorkbook;
      currentSourceWorkbook = edited.sourceWorkbook;
      files.push({
        label: `Скачать ${result.blankLabel || "бланк"}`,
        name: currentBlankOutputNames.get(result.blankId) || "blank заполненный.xlsx",
        blob: new Blob([saveXlsx(result.blankWorkbook)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      });
    }

    const sourceBytes = saveXlsx(currentSourceWorkbook);
    files.push({
      label: "Скачать таблицу заказа",
      name: currentSourceOutputName,
      blob: new Blob([sourceBytes], {
        type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      }),
    });
    prepareDownloadLinks(files);
    statusEl.textContent = "Файлы готовы";
  } catch (error) {
    statusEl.textContent = "Ошибка";
    alert(error.message || "Не удалось сохранить правки.");
  } finally {
    downloadButton.disabled = false;
  }
});

issueReportButton.addEventListener("click", downloadIssueReport);

function handleReportInput(event) {
  if (event.target.matches(".qty-input, .comment-input")) {
    const key = event.target.dataset.key;
    const current = editState.get(key) || { value: "", comment: "" };
    if (event.target.matches(".qty-input")) current.value = event.target.value;
    if (event.target.matches(".comment-input")) current.comment = event.target.value;
    editState.set(key, current);
    const row = event.target.closest("tr");
    row?.classList.remove("invalid");
    if (row) updateCommentHint(row);
  }
}

reportBody.addEventListener("input", handleReportInput);
priorityBody.addEventListener("input", handleReportInput);

metricsEl.addEventListener("click", (event) => {
  const metric = event.target.closest(".metric");
  if (!metric) return;
  const nextFilter = metric.dataset.filter;
  activeFilter = activeFilter === nextFilter ? null : nextFilter;
  renderMetrics(combinedSummary(currentResults));
  renderReportView();
});

reportSearch.addEventListener("input", renderReportView);

clearFilterButton.addEventListener("click", () => {
  activeFilter = null;
  reportSearch.value = "";
  renderMetrics(combinedSummary(currentResults));
  renderReportView();
});
