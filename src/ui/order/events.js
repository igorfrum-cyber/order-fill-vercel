import { priceOrderRows } from "./../../domain/budget/pricing.js";
import { fillWorkbook } from "./../../workflows/order/fill.js";
import { outputFileName, sourceOutputFileName } from "./../../shared/filenames.js";
import { mergeTyumenSources } from "./../../workflows/tyumen/merge.js";
import { budgetReportRows } from "./../../workflows/budget/adapters.js";
import { applyFinalEdits } from "./../../workflows/order/edits.js";
import { addOrderBlankStocks } from "./../../infrastructure/excel/stocks.js";
import { saveXlsx } from "./../../infrastructure/excel/archive.js";
import { applyBudgetWorkbookPricing } from "./../../infrastructure/excel/pricing.js";

export function bindDocumentKeydown(context) {
  const {elements, state, actions} = context;
document.addEventListener("keydown", (event) => {
  if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
  if (event.altKey || event.ctrlKey || event.metaKey) return;
  if (event.target?.matches?.("select")) return;

  const target = actions.scrollTargetForKeyboard(event.target);
  const delta = event.key === "ArrowDown" ? 72 : -72;
  event.preventDefault();
  target.scrollBy({ top: delta, behavior: "auto" });
});
}

export function bindTwoTyumenSourcesChange(context) {
  const {elements, state, actions} = context;
elements.twoTyumenSources.addEventListener("change", () => {
  document.querySelector("#warehouseSourceField").classList.toggle("hidden", !elements.twoTyumenSources.checked);
  document.querySelector("#tyumenUploadCell").classList.toggle("enabled", elements.twoTyumenSources.checked);
  elements.twoTyumenSources.setAttribute("aria-expanded", String(elements.twoTyumenSources.checked));
  elements.warehouseSourceFile.required = elements.twoTyumenSources.checked;
  elements.warehouseSourceFile.disabled = !elements.twoTyumenSources.checked;
  if (!elements.twoTyumenSources.checked) {
    elements.warehouseSourceFile.value = "";
    document.querySelector("#warehouseSourceName").textContent = ".xlsx, .xlsm или .xls";
  }
  elements.sourceFile.closest("label").querySelector(".label").textContent = elements.twoTyumenSources.checked ? "Офис: Склад Тюмень" : "Таблица заказа товара";
  actions.resetFillState();
});
}

export function bindFormSubmit(context) {
  const {elements, state, actions} = context;
elements.form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const brand = actions.selectedBrand();
  const blankInputs = brand === "christina"
    ? [
        { id: "home", label: "HOME", file: elements.homeFile.files[0] },
        { id: "proff", label: "PROFF", file: elements.proffFile.files[0] },
      ]
    : [{ id: "main", label: actions.mainBlankLabelForBrand(brand), file: elements.blankFile.files[0] }];
  if (!elements.sourceFile.files[0] || blankInputs.some((item) => !item.file)) return;
  let priceSettings;
  try {priceSettings=state.orderPricing.read();}catch(e){alert(e.message);return;}

  elements.statusEl.textContent = "Обработка...";
  actions.setSubmitButtonState("processing");
  elements.downloadButton.disabled = true;
  elements.issueReportButton.disabled = true;
  elements.resultEl.classList.add("hidden");
  actions.clearDownloadLinks();
  state.isFormFilled = false;
  state.currentResults = [];
  state.currentReportRows = [];
  state.activeFilter = null;
  state.editState = new Map();
  state.currentBlankWorkbooks = new Map();
  state.currentBlankOutputNames = new Map();
  state.currentSourceWorkbook = null;

  try {
    let sourceWorkbook = await actions.loadWorkbook(elements.sourceFile.files[0]);
    if (elements.twoTyumenSources.checked) actions.assertDifferentTyumenFiles(elements.sourceFile.files[0], elements.warehouseSourceFile.files[0]);
    if (elements.twoTyumenSources.checked) sourceWorkbook = mergeTyumenSources({
      officeWorkbook: sourceWorkbook,
      warehouseWorkbook: await actions.loadWorkbook(elements.warehouseSourceFile.files[0]),
      officeFileName: elements.sourceFile.files[0].name,
      warehouseFileName: elements.warehouseSourceFile.files[0].name,
      brand,
    });
    const blankWorkbooks = await Promise.all(blankInputs.map((item) => actions.loadWorkbook(item.file, { allowLegacyXls: brand === "novacutan" })));
    const results = blankInputs.map((item, index) => fillWorkbook({
      sourceWorkbook,
      sourceFileName: elements.sourceFile.files[0].name,
      blankWorkbook: blankWorkbooks[index],
      orderMonth: elements.orderMonth.value,
      brand,
      blankId: item.id,
      blankLabel: item.label,
    }));

    state.currentResults = results;
    for(const result of results){result.priceSettings=priceSettings;result.budgetPricing=priceOrderRows(budgetReportRows(result,brand),priceSettings);}
    state.currentSourceWorkbook = sourceWorkbook;
    state.currentBlankWorkbooks = new Map(results.map((result) => [result.blankId, result.blankWorkbook]));
    for (const [index, item] of blankInputs.entries()) {
      state.currentBlankWorkbooks.set(item.id, results[index].blankWorkbook);
      state.currentBlankOutputNames.set(item.id, outputFileName(item.file.name, results[index].summary.sourceCity));
    }
    state.currentSourceOutputName = elements.twoTyumenSources.checked ? `${elements.brandSelect.selectedOptions[0].textContent} Тюмень общая заполненная таблица.xlsx` : sourceOutputFileName(elements.sourceFile.files[0].name);

    const rows = [...results.flatMap((result) => result.reportRows), ...actions.sourceDuplicateRows(results), ...actions.missingInBlankRows(results)];
    state.currentReportRows = rows;
    state.editState = new Map(rows.map((row) => [row.key || `${row.blankId}:${row.blankRow}`, { value: row.inserted ?? "", comment: actions.initialComment(row) }]));
    for (const row of rows) if (row.hasOrderedFact && Number(row.orderedFact) === 0) state.budgetLocks.add(row.key);
    actions.renderMetrics(actions.combinedSummary(results));
    actions.renderReportView();
    actions.refreshBudgetTotals();
    elements.resultEl.classList.remove("hidden");
    elements.downloadButton.disabled = false;
    elements.issueReportButton.disabled = !actions.issueReportRows().length;
    elements.statusEl.textContent = "Готово";
    state.isFormFilled = true;
    actions.setSubmitButtonState("completed");
  } catch (error) {
    elements.statusEl.textContent = "Ошибка";
    actions.setSubmitButtonState("ready");
    alert(error.message || "Не удалось обработать файлы.");
  } finally {
    if (!state.isFormFilled) actions.setSubmitButtonState("ready");
  }
});
}

export function bindDownloadButtonClick(context) {
  const {elements, state, actions} = context;
elements.downloadButton.addEventListener("click", async () => {
  if (!state.currentResults.length || !state.currentBlankWorkbooks.size || !state.currentSourceWorkbook) {
    alert("Сначала заполните бланк.");
    return;
  }
  if (!actions.validateEdits()) return;
  if (!actions.confirmQualityWarnings()) return;

  elements.downloadButton.disabled = true;
  elements.statusEl.textContent = "Сохраняю правки...";
  try {
    const edits = actions.collectEdits();
    const files = [];
    for (const result of state.currentResults) {
      const edited = applyFinalEdits({
        blankWorkbook: result.blankWorkbook,
        sourceWorkbook: state.currentSourceWorkbook,
        reportRows: result.reportRows,
        edits,
        brand: actions.selectedBrand(),
      });
      result.blankWorkbook = edited.blankWorkbook;
      state.currentSourceWorkbook = edited.sourceWorkbook;
      result.budgetPricing=priceOrderRows(actions.liveBudgetRows(false).filter(r=>r.group===result.blankId),result.priceSettings);
      const pricedBlank = applyBudgetWorkbookPricing(
        result.blankWorkbook, result.blankDetection.sheetName, result.blankDetection.headerRow, result.budgetPricing,
      );
      const outputBlank = elements.includeBlankStocks.checked
        ? addOrderBlankStocks(pricedBlank, result.blankDetection, result.reportRows, actions.selectedBrand())
        : pricedBlank;
      files.push({
        label: `Скачать ${result.blankLabel || "бланк"}`,
        name: state.currentBlankOutputNames.get(result.blankId) || "blank заполненный.xlsx",
        blob: new Blob([saveXlsx(outputBlank)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      });
    }

    const sourceBytes = saveXlsx(state.currentSourceWorkbook);
    files.push({
      label: "Скачать таблицу заказа",
      name: state.currentSourceOutputName,
      blob: new Blob([sourceBytes], {
        type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      }),
    });
    actions.prepareDownloadLinks(files);
    elements.statusEl.textContent = "Файлы готовы";
  } catch (error) {
    elements.statusEl.textContent = "Ошибка";
    alert(error.message || "Не удалось сохранить правки.");
  } finally {
    elements.downloadButton.disabled = false;
  }
});
}

export function bindMetricsElClick(context) {
  const {elements, state, actions} = context;
elements.metricsEl.addEventListener("click", (event) => {
  const metric = event.target.closest(".metric");
  if (!metric) return;
  const nextFilter = metric.dataset.filter;
  state.activeFilter = state.activeFilter === nextFilter ? null : nextFilter;
  actions.renderMetrics(actions.combinedSummary(state.currentResults));
  actions.renderReportView();
});
}

export function bindClearFilterButtonClick(context) {
  const {elements, state, actions} = context;
elements.clearFilterButton.addEventListener("click", () => {
  state.activeFilter = null;
  elements.reportSearch.value = "";
  actions.renderMetrics(actions.combinedSummary(state.currentResults));
  actions.renderReportView();
});
}
