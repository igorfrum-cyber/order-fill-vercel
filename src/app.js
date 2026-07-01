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
const orderMonth = document.querySelector("#orderMonth");
const sourceFile = document.querySelector("#sourceFile");
const blankFile = document.querySelector("#blankFile");
const sourceName = document.querySelector("#sourceName");
const blankName = document.querySelector("#blankName");
const resultEl = document.querySelector("#result");
const metricsEl = document.querySelector("#metrics");
const periodNote = document.querySelector("#periodNote");
const reportBody = document.querySelector("#reportBody");
const downloadButton = document.querySelector("#downloadButton");
const downloadLinks = document.querySelector("#downloadLinks");
const blankDownloadLink = document.querySelector("#blankDownloadLink");
const sourceDownloadLink = document.querySelector("#sourceDownloadLink");
const submitButton = form.querySelector("button");

let currentResult = null;
let currentBlankWorkbook = null;
let currentSourceWorkbook = null;
let currentBlankOutputName = "blank заполненный.xlsx";
let currentSourceOutputName = "order заполненная таблица.xlsx";
let currentDownloadUrls = [];

function setDefaultOrderMonth() {
  const date = new Date();
  date.setMonth(date.getMonth() + 1);
  orderMonth.value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

setDefaultOrderMonth();

function bindFileName(input, output) {
  input.addEventListener("change", () => {
    output.textContent = input.files[0]?.name || ".xlsx или .xlsm";
  });
}

bindFileName(sourceFile, sourceName);
bindFileName(blankFile, blankName);

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
  periodNote.textContent = `Заказ на ${summary.orderMonthLabel}. Период: ${summary.actualMainPeriod}. Прошлый период: ${summary.actualPreviousPeriod}.`;
}

function renderReport(rows) {
  reportBody.innerHTML = rows
    .map((row) => {
      const cls = row.status === "warning_name_differs" || row.status === "warning_name_only" ? "warn" : row.status === "matched" || row.status === "matched_by_name" ? "ok" : "muted";
      const inserted = row.inserted ?? "";
      const comment = row.autoComment || "";
      return `
        <tr>
          <td class="${cls}">${statusLabel(row.status)}</td>
          <td>${escapeHtml(row.blankArticle)}</td>
          <td>${escapeHtml(row.blankName)}</td>
          <td>${escapeHtml(row.blankUnit)}</td>
          <td>${escapeHtml(row.stock ?? "")}</td>
          <td>${escapeHtml(row.inTransit ?? "")}</td>
          <td>${Number(row.recommended).toFixed(2)}</td>
          <td>
            <input
              class="qty-input"
              type="number"
              min="0"
              step="1"
              inputmode="numeric"
              data-row="${row.blankRow}"
              data-initial-value="${inserted}"
              data-auto-comment="${escapeHtml(row.autoComment || "")}"
              value="${escapeHtml(inserted)}"
              aria-label="Количество для строки ${row.blankRow}"
            />
          </td>
          <td>
            <input
              class="comment-input"
              type="text"
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
}

async function loadWorkbook(file) {
  const buffer = await file.arrayBuffer();
  return loadXlsx(buffer);
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!sourceFile.files[0] || !blankFile.files[0]) return;

  statusEl.textContent = "Обработка...";
  submitButton.disabled = true;
  downloadButton.disabled = true;
  resultEl.classList.add("hidden");
  clearDownloadLinks();
  currentResult = null;
  currentBlankWorkbook = null;
  currentSourceWorkbook = null;

  try {
    const [sourceWorkbook, blankWorkbook] = await Promise.all([
      loadWorkbook(sourceFile.files[0]),
      loadWorkbook(blankFile.files[0]),
    ]);
    const result = fillWorkbook({
      sourceWorkbook,
      blankWorkbook,
      orderMonth: orderMonth.value,
    });

    currentResult = result;
    currentBlankWorkbook = result.blankWorkbook;
    currentSourceWorkbook = result.sourceWorkbook;
    currentBlankOutputName = outputFileName(blankFile.files[0].name);
    currentSourceOutputName = sourceOutputFileName(sourceFile.files[0].name);

    renderMetrics(result.summary);
    renderReport(result.reportRows);
    resultEl.classList.remove("hidden");
    downloadButton.disabled = false;
    statusEl.textContent = "Готово";
  } catch (error) {
    statusEl.textContent = "Ошибка";
    alert(error.message || "Не удалось обработать файлы.");
  } finally {
    submitButton.disabled = false;
  }
});

function collectEdits() {
  const comments = new Map(Array.from(document.querySelectorAll(".comment-input")).map((input) => [Number(input.dataset.row), input.value]));
  return Array.from(document.querySelectorAll(".qty-input")).map((input) => ({
    blankRow: Number(input.dataset.row),
    value: input.value,
    comment: comments.get(Number(input.dataset.row)) || "",
  }));
}

function clearDownloadLinks() {
  for (const url of currentDownloadUrls) URL.revokeObjectURL(url);
  currentDownloadUrls = [];
  downloadLinks.classList.add("hidden");
  blankDownloadLink.removeAttribute("download");
  sourceDownloadLink.removeAttribute("download");
  blankDownloadLink.href = "#";
  sourceDownloadLink.href = "#";
}

function validateEdits() {
  let invalidCount = 0;
  for (const row of reportBody.querySelectorAll("tr")) row.classList.remove("invalid");

  for (const input of document.querySelectorAll(".qty-input")) {
    const row = input.closest("tr");
    const commentInput = row.querySelector(".comment-input");
    const initial = input.dataset.initialValue === "" ? null : Number(input.dataset.initialValue);
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
    const changed = value !== initial;
    const stillAutoComment = autoComment && comment.toLowerCase() === autoComment;
    if (changed && (!comment || stillAutoComment)) {
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
  const autoComment = (qtyInput.dataset.autoComment || "").trim().toLowerCase();
  let value;
  try {
    value = normalizeOrderValue(qtyInput.value);
  } catch {
    return false;
  }

  const changed = value !== initial;
  const comment = commentInput.value.trim();
  const stillAutoComment = autoComment && comment.toLowerCase() === autoComment;
  return changed && (!comment || stillAutoComment);
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

function prepareDownloadLinks(blankBlob, sourceBlob) {
  clearDownloadLinks();
  const blankUrl = URL.createObjectURL(blankBlob);
  const sourceUrl = URL.createObjectURL(sourceBlob);
  currentDownloadUrls = [blankUrl, sourceUrl];

  blankDownloadLink.href = blankUrl;
  blankDownloadLink.download = currentBlankOutputName;
  sourceDownloadLink.href = sourceUrl;
  sourceDownloadLink.download = currentSourceOutputName;
  downloadLinks.classList.remove("hidden");

  triggerDownload(blankUrl, currentBlankOutputName);
  window.setTimeout(() => triggerDownload(sourceUrl, currentSourceOutputName), 250);
}

downloadButton.addEventListener("click", async () => {
  if (!currentResult || !currentBlankWorkbook || !currentSourceWorkbook) {
    alert("Сначала заполните бланк.");
    return;
  }
  if (!validateEdits()) return;

  downloadButton.disabled = true;
  statusEl.textContent = "Сохраняю правки...";
  try {
    const edited = applyFinalEdits({
      blankWorkbook: currentBlankWorkbook,
      sourceWorkbook: currentSourceWorkbook,
      reportRows: currentResult.reportRows,
      edits: collectEdits(),
    });
    currentBlankWorkbook = edited.blankWorkbook;
    currentSourceWorkbook = edited.sourceWorkbook;

    const bytes = saveXlsx(currentBlankWorkbook);
    const sourceBytes = saveXlsx(currentSourceWorkbook);
    const blankBlob = new Blob([bytes], {
      type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    });
    const sourceBlob = new Blob([sourceBytes], {
      type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    });
    prepareDownloadLinks(blankBlob, sourceBlob);
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
