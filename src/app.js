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
const adjustmentHeader = document.querySelector("#adjustmentHeader");
const downloadButton = document.querySelector("#downloadButton");
const downloadLinks = document.querySelector("#downloadLinks");
const submitButton = form.querySelector("button");

let currentResults = [];
let currentBlankWorkbooks = new Map();
let currentSourceWorkbook = null;
let currentBlankOutputNames = new Map();
let currentSourceOutputName = "order заполненная таблица.xlsx";
let currentDownloadUrls = [];
let isFormFilled = false;

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
  currentBlankWorkbooks = new Map();
  currentBlankOutputNames = new Map();
  currentSourceWorkbook = null;
  resultEl.classList.add("hidden");
  downloadButton.disabled = true;
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
  };
}

function renderMetrics(summary) {
  const rows = [
    ["Заполнено", summary.filled],
    ["Оставлено пустым", summary.leftBlank],
    ["Проверить", summary.suspicious],
    ["Не найдено", summary.unmatched],
    ["Дублей", summary.duplicates],
  ];
  metricsEl.innerHTML = rows
    .map(([label, value]) => `<div class="metric"><strong>${value}</strong><span>${label}</span></div>`)
    .join("");
  periodNote.textContent = `${summary.brand}. Заказ на ${summary.orderMonthLabel}. Период: ${summary.actualMainPeriod}. Прошлый период: ${summary.actualPreviousPeriod}.`;
}

function renderReport(rows) {
  reportBody.innerHTML = rows
    .map((row) => {
      const cls = row.status === "warning_name_differs" || row.status === "warning_name_only" ? "warn" : row.status === "matched" || row.status === "matched_by_name" ? "ok" : "muted";
      const inserted = row.inserted ?? "";
      const comment = row.sourceComment || row.autoComment || "";
      const baseline = Number(row.recommended) < 1.5 || Number(row.rounded) <= 0 ? "" : row.rounded;
      const rowKey = row.key || `${row.blankId}:${row.blankRow}`;
      return `
        <tr>
          <td class="${cls}">${statusLabel(row.status)}</td>
          <td>${escapeHtml(row.blankLabel)}</td>
          <td>${escapeHtml(row.blankArticle)}</td>
          <td>${escapeHtml(row.blankName)}</td>
          <td>${escapeHtml(row.blankUnit)}</td>
          <td>${escapeHtml(row.stock ?? "")}</td>
          <td>${escapeHtml(row.inTransit ?? "")}</td>
          <td>${Number(row.recommended).toFixed(2)}</td>
          <td>${row.hasOrderedFact ? escapeHtml(row.orderedFact) : ""}</td>
          <td>${escapeHtml(row.blankBoxSize ?? "")}</td>
          <td>
            <input
              class="qty-input"
              type="number"
              min="0"
              step="1"
              inputmode="numeric"
              data-key="${escapeHtml(rowKey)}"
              data-blank-id="${escapeHtml(row.blankId)}"
              data-row="${row.blankRow}"
              data-initial-value="${inserted}"
              data-baseline-value="${baseline}"
              data-auto-comment="${escapeHtml(row.autoComment || "")}"
              value="${escapeHtml(inserted)}"
              aria-label="Количество для строки ${row.blankRow}"
            />
          </td>
          <td>
            <input
              class="comment-input"
              type="text"
              data-key="${escapeHtml(rowKey)}"
              data-blank-id="${escapeHtml(row.blankId)}"
              data-row="${row.blankRow}"
              value="${escapeHtml(comment)}"
              aria-label="Комментарий для строки ${row.blankRow}"
            />
          </td>
          <td>${Math.round(Number(row.similarity || 0) * 100)}%</td>
        </tr>
      `;
    })
    .join("");
  for (const row of reportBody.querySelectorAll("tr")) updateCommentHint(row);
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
  resultEl.classList.add("hidden");
  clearDownloadLinks();
  isFormFilled = false;
  currentResults = [];
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
    renderMetrics(combinedSummary(results));
    renderReport(rows);
    resultEl.classList.remove("hidden");
    downloadButton.disabled = false;
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
  const comments = new Map(Array.from(document.querySelectorAll(".comment-input")).map((input) => [input.dataset.key, input.value]));
  return Array.from(document.querySelectorAll(".qty-input")).map((input) => ({
    key: input.dataset.key,
    blankId: input.dataset.blankId,
    blankRow: Number(input.dataset.row),
    value: input.value,
    comment: comments.get(input.dataset.key) || "",
  }));
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

  for (const input of document.querySelectorAll(".qty-input")) {
    const row = input.closest("tr");
    const commentInput = row.querySelector(".comment-input");
    const initial = input.dataset.initialValue === "" ? null : Number(input.dataset.initialValue);
    const baseline = input.dataset.baselineValue === "" ? null : Number(input.dataset.baselineValue);
    const autoComment = (input.dataset.autoComment || "").trim().toLowerCase();
    let value;
    try {
      value = normalizeOrderValue(input.value);
    } catch {
      row.classList.add("invalid");
      invalidCount += 1;
      continue;
    }
    const comment = commentInput.value.trim();
    const requiresComment = value !== baseline;
    const stillAutoComment = autoComment && comment.toLowerCase() === autoComment;
    const autoCommentAllowed = stillAutoComment && value === initial;
    if (requiresComment && (!comment || (stillAutoComment && !autoCommentAllowed))) {
      row.classList.add("invalid");
      invalidCount += 1;
    }
  }

  if (invalidCount > 0) {
    const firstInvalid = reportBody.querySelector("tr.invalid");
    firstInvalid?.scrollIntoView({ block: "center", behavior: "smooth" });
    alert("Есть строки, где изменено значение «Вставлено», но не заполнен новый комментарий.");
    return false;
  }
  return true;
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

reportBody.addEventListener("input", (event) => {
  if (event.target.matches(".qty-input, .comment-input")) {
    const row = event.target.closest("tr");
    row?.classList.remove("invalid");
    if (row) updateCommentHint(row);
  }
});
