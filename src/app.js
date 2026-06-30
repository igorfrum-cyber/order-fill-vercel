import "./styles.css";
import { applyEdits, fillWorkbook, loadXlsx, outputFileName, saveXlsx } from "./workbookProcessor.js";

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
const submitButton = form.querySelector("button");

let currentResult = null;
let currentBlankWorkbook = null;
let currentOutputName = "blank заполненный.xlsx";

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
    .slice(0, 500)
    .map((row) => {
      const cls = row.status === "warning_name_differs" || row.status === "warning_name_only" ? "warn" : row.status === "matched" || row.status === "matched_by_name" ? "ok" : "muted";
      const inserted = row.status === "matched" || row.status === "matched_by_name" ? row.rounded : "";
      return `
        <tr>
          <td class="${cls}">${statusLabel(row.status)}</td>
          <td>${escapeHtml(row.blankArticle)}</td>
          <td>${escapeHtml(row.blankName)}</td>
          <td>${Number(row.recommended).toFixed(2)}</td>
          <td>
            <input
              class="qty-input"
              type="number"
              min="0"
              step="1"
              inputmode="numeric"
              data-row="${row.blankRow}"
              value="${inserted}"
              aria-label="Количество для строки ${row.blankRow}"
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
  currentResult = null;
  currentBlankWorkbook = null;

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
    currentOutputName = outputFileName(blankFile.files[0].name);

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
  return Array.from(document.querySelectorAll(".qty-input")).map((input) => ({
    blankRow: Number(input.dataset.row),
    value: input.value,
  }));
}

function downloadBlob(blob, fileName) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

downloadButton.addEventListener("click", async () => {
  if (!currentResult || !currentBlankWorkbook) {
    alert("Сначала заполните бланк.");
    return;
  }
  downloadButton.disabled = true;
  statusEl.textContent = "Сохраняю правки...";
  try {
    applyEdits(currentBlankWorkbook, collectEdits());
    const bytes = saveXlsx(currentBlankWorkbook);
    const blob = new Blob([bytes], {
      type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    });
    downloadBlob(blob, currentOutputName);
    statusEl.textContent = "Файл готов";
  } catch (error) {
    statusEl.textContent = "Ошибка";
    alert(error.message || "Не удалось сохранить правки.");
  } finally {
    downloadButton.disabled = false;
  }
});
