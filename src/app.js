import "./styles.css";
import {
  applyFinalEdits,
  buildNorthOrderFiles,
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
const orderSection = document.querySelector("#orderSection");
const northSection = document.querySelector("#northSection");
const orderModeButton = document.querySelector("#orderModeButton");
const northModeButton = document.querySelector("#northModeButton");
const northForm = document.querySelector("#northForm");
const northFiles = document.querySelector("#northFiles");
const northNames = document.querySelector("#northNames");
const northStatus = document.querySelector("#northStatus");
const northResult = document.querySelector("#northResult");
const northSummary = document.querySelector("#northSummary");
const northDownloadLinks = document.querySelector("#northDownloadLinks");
const northSubmitButton = northForm.querySelector("button");
const northBackButton = document.querySelector("#northBackButton");

let currentResults = [];
let currentReportRows = [];
let currentBlankWorkbooks = new Map();
let currentSourceWorkbook = null;
let currentBlankOutputNames = new Map();
let currentSourceOutputName = "order заполненная таблица.xlsx";
let currentDownloadUrls = [];
let currentNorthDownloadUrls = [];
let isFormFilled = false;
let activeFilter = null;
let editState = new Map();

function setDefaultOrderMonth() {
  const date = new Date();
  date.setMonth(date.getMonth() + 1);
  orderMonth.value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

setDefaultOrderMonth();

function setActiveMode(mode) {
  const isNorth = mode === "north";
  orderSection.classList.toggle("hidden", isNorth);
  northSection.classList.toggle("hidden", !isNorth);
  orderModeButton.classList.toggle("active", !isNorth);
  northModeButton.classList.toggle("active", isNorth);
  orderModeButton.setAttribute("aria-pressed", String(!isNorth));
  northModeButton.setAttribute("aria-pressed", String(isNorth));
}

orderModeButton.addEventListener("click", () => setActiveMode("order"));
northModeButton.addEventListener("click", () => setActiveMode("north"));
northBackButton.addEventListener("click", () => setActiveMode("order"));
setActiveMode("order");

function scrollTargetForKeyboard(element) {
  return element?.closest?.(".table-wrap, .priority-wrap") || document.scrollingElement || document.documentElement;
}

document.addEventListener("keydown", (event) => {
  if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
  if (event.altKey || event.ctrlKey || event.metaKey) return;
  if (event.target?.matches?.("select")) return;

  const target = scrollTargetForKeyboard(event.target);
  const delta = event.key === "ArrowDown" ? 72 : -72;
  event.preventDefault();
  target.scrollBy({ top: delta, behavior: "auto" });
});

function bindFileName(input, output, placeholder = ".xlsx, .xlsm или .xls") {
  input.addEventListener("change", () => {
    output.textContent = input.files[0]?.name || placeholder;
    resetFillState();
  });
}

bindFileName(sourceFile, sourceName);
bindFileName(blankFile, blankName, ".xlsx, .xlsm или .xls");
bindFileName(homeFile, homeName, ".xlsx или .xlsm");
bindFileName(proffFile, proffName, ".xlsx или .xlsm");

northFiles.addEventListener("change", () => {
  const files = Array.from(northFiles.files || []);
  northNames.textContent = files.length ? files.map((file) => file.name).join(", ") : "Городские заполненные бланки .xlsx, .xlsm или .xls";
  clearNorthDownloadLinks();
  northResult.classList.add("hidden");
  northStatus.textContent = "Готов к загрузке";
});

function selectedBrand() {
  return brandSelect.value || "angiopharm";
}

function adjustmentLabelForBrand(brand) {
  if (brand === "christina") return "Кратность";
  if (brand === "levissime") return "Кол-во в уп.";
  if (brand === "sothys") return "Округление";
  if (brand === "novacutan") return "Мин. заказ";
  return "Шт. в коробке";
}

function mainBlankLabelForBrand(brand) {
  if (brand === "levissime") return "LeviSsime";
  if (brand === "sothys") return "SOTHYS";
  if (brand === "novacutan") return "NOVACUTAN";
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
    notInBlank: currentReportRows.filter((row) => row.status === "not_in_blank").length,
    duplicates: currentReportRows.filter((row) => row.duplicate).length,
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

async function loadWorkbook(file, options = {}) {
  const buffer = await file.arrayBuffer();
  return loadXlsx(await normalizeWorkbookBytes(buffer, file.name, options));
}

function isLegacyXls(fileName) {
  return /\.xls$/i.test(fileName) && !/\.xlsx$/i.test(fileName) && !/\.xlsm$/i.test(fileName);
}

async function normalizeWorkbookBytes(buffer, fileName, options = {}) {
  if (!isLegacyXls(fileName)) return buffer;
  if (options.allowLegacyXls === false) {
    throw new Error(`Файл «${fileName}» в старом формате .xls нельзя использовать как бланк: при такой конвертации теряется оформление. Откройте этот бланк в Excel и сохраните как .xlsx, затем загрузите .xlsx.`);
  }
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
    const blankWorkbooks = await Promise.all(blankInputs.map((item) => loadWorkbook(item.file, { allowLegacyXls: brand === "novacutan" })));
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

    const rows = [...results.flatMap((result) => result.reportRows), ...sourceDuplicateRows(results), ...missingInBlankRows(results)];
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
  return currentReportRows.filter((row) => row.status === "warning_name_differs" || row.status === "warning_name_only" || row.status === "not_in_source" || row.duplicate);
}

function issueReason(row) {
  const reasons = [];
  if (row.status === "warning_name_only") reasons.push("В таблице заказа нет артикула, найдено только по названию");
  if (row.status === "warning_name_differs") reasons.push("Артикул найден, но название сильно отличается");
  if (row.status === "not_in_source") reasons.push("Позиция есть в бланке, но не найдена в таблице заказа");
  if (row.status === "source_duplicate") reasons.push("В таблице заказа есть несколько строк с одним артикулом");
  if (row.duplicate) reasons.push("Есть дублирующиеся кандидаты по артикулу");
  const duplicateText = duplicateDescription(row);
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
    const edit = rowEdit(row);
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
  return [header, ...rows].map((row) => row.map(csvCell).join(";")).join("\n");
}

function clearDownloadLinks() {
  for (const url of currentDownloadUrls) URL.revokeObjectURL(url);
  currentDownloadUrls = [];
  downloadLinks.classList.add("hidden");
  downloadLinks.innerHTML = "";
}

function clearNorthDownloadLinks() {
  for (const url of currentNorthDownloadUrls) URL.revokeObjectURL(url);
  currentNorthDownloadUrls = [];
  northDownloadLinks.classList.add("hidden");
  northDownloadLinks.innerHTML = "";
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
  const notInSourceCount = currentReportRows.filter((row) => row.status === "not_in_source").length;
  const notInBlankCount = currentReportRows.filter((row) => row.status === "not_in_blank").length;
  const manualCount = currentReportRows.filter(isManualDeviation).length;
  const blankDuplicateCount = currentResults.reduce((sum, result) => sum + (result.summary.blankDuplicateArticles || 0), 0);
  const total = issueCount + duplicateCount + notInSourceCount + notInBlankCount + manualCount + blankDuplicateCount;
  if (!total) return true;

  const lines = [
    `Проверьте ${total} спорных строк/ситуаций перед скачиванием.`,
    issueCount ? `Проверить: ${issueCount}` : "",
    duplicateCount ? `Дубли: ${duplicateCount}` : "",
    notInSourceCount ? `Нет в таблице: ${notInSourceCount}` : "",
    notInBlankCount ? `Нет в бланке: ${notInBlankCount}` : "",
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

function todayRu() {
  const date = new Date();
  return `${String(date.getDate()).padStart(2, "0")}.${String(date.getMonth() + 1).padStart(2, "0")}.${date.getFullYear()}`;
}

function transferWorkbookBytes(transfer) {
  return import("xlsx").then(({ utils, write }) => {
    const rows = [
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", `Заказ на перемещение от ${todayRu()}`, "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "Отправитель:", "Склад Тюмень", "Получатель:", transfer.city.warehouse, "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "№", "Товар", "Количество", "", "", "", ""],
      ...transfer.items.map((item, index) => ["", index + 1, item.name, item.quantity, item.unit || "шт", "", "", ""]),
      ["", "", "", "", "", "", "", ""],
      ["Менеджер", "", "", "", "", "", "", ""],
    ];
    const workbook = utils.book_new();
    const sheet = utils.aoa_to_sheet(rows);
    sheet["!cols"] = [
      { wch: 8 },
      { wch: 8 },
      { wch: 72 },
      { wch: 14 },
      { wch: 10 },
      { wch: 22 },
      { wch: 12 },
      { wch: 12 },
    ];
    utils.book_append_sheet(workbook, sheet, "Лист_1");
    return write(workbook, { bookType: "xlsx", type: "array" });
  });
}

function prepareNorthDownloadLinks(files) {
  clearNorthDownloadLinks();
  currentNorthDownloadUrls = files.map((file) => URL.createObjectURL(file.blob));
  northDownloadLinks.innerHTML = files
    .map((file, index) => `<a class="file-link" href="${currentNorthDownloadUrls[index]}" download="${escapeHtml(file.name)}">${escapeHtml(file.label)}</a>`)
    .join("");
  northDownloadLinks.classList.remove("hidden");
  currentNorthDownloadUrls.forEach((url, index) => {
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

northForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const files = Array.from(northFiles.files || []);
  if (!files.length) return;

  northSubmitButton.disabled = true;
  northStatus.textContent = "Соединяю...";
  clearNorthDownloadLinks();
  northResult.classList.add("hidden");

  try {
    const blanks = await Promise.all(files.map(async (file) => ({
      fileName: file.name,
      workbook: await loadWorkbook(file),
    })));
    const result = buildNorthOrderFiles(blanks);
    const outputFiles = [
      {
        label: "Скачать общий бланк",
        name: result.summaryFileName,
        blob: new Blob([saveXlsx(result.summaryWorkbook)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      },
    ];
    for (const transfer of result.transfers) {
      outputFiles.push({
        label: `Скачать перемещение ${transfer.city.label}`,
        name: transfer.fileName,
        blob: new Blob([await transferWorkbookBytes(transfer)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      });
    }

    northSummary.textContent = `Города: ${result.uploadedCities.join(", ")}. В общем бланке позиций с заказом: ${result.totalsCount}. Перемещений: ${result.transfers.length}.${result.appendedToSummary.length ? ` Добавлено в конец общего бланка: ${result.appendedToSummary.length}.` : ""}${result.adjustedToMinimum ? ` До минимальной партии подтянуто: ${result.adjustedToMinimum}.` : ""}`;
    prepareNorthDownloadLinks(outputFiles);
    northResult.classList.remove("hidden");
    northStatus.textContent = "Файлы готовы";
  } catch (error) {
    northStatus.textContent = "Ошибка";
    alert(error.message || "Не удалось соединить бланки.");
  } finally {
    northSubmitButton.disabled = false;
  }
});

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
