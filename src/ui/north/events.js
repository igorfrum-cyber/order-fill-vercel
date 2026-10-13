import { validateNorthTyumenSourceWorkbook } from "./../../infrastructure/excel/cities.js";
import { priceOrderRows } from "./../../domain/budget/pricing.js";
import { buildNorthOrderFiles, finalizeNorthOrderFiles } from "./../../workflows/north/order.js";
import { mergeTyumenSources } from "./../../workflows/tyumen/merge.js";
import { saveXlsx } from "./../../infrastructure/excel/archive.js";

export function bindNorthTwoSourcesChange(context) {
  const {elements, state, actions} = context;
elements.northTwoSources.addEventListener("change", () => {
  document.querySelector("#northWarehouseField").classList.toggle("hidden", !elements.northTwoSources.checked);
  elements.northWarehouseFile.required = elements.northTwoSources.checked;
  elements.northSourceFile.required = elements.northTwoSources.checked;
  elements.northSourceFile.closest("label").querySelector(".label").textContent = elements.northTwoSources.checked ? "Офис: Склад Тюмень" : "Заполненная таблица Тюмени";
  actions.resetNorthCalculationState();
});
}

export function bindNorthFileInputChange(context) {
  const {elements, state, actions} = context;
elements.northFileInput.addEventListener("change", () => {
  actions.addPendingNorthFiles({
    input: elements.northFileInput,
    names: elements.northNames,
    placeholder: "Выберите один или несколько заполненных бланков",
    files: state.addedNorthFiles,
    list: elements.northFileList,
    group: "default",
  });
});
}

export function bindNorthHomeInputChange(context) {
  const {elements, state, actions} = context;
elements.northHomeInput.addEventListener("change", () => {
  actions.addPendingNorthFiles({
    input: elements.northHomeInput,
    names: elements.northHomeNames,
    placeholder: "Выберите HOME-бланки городов",
    files: state.addedNorthHomeFiles,
    list: elements.northHomeFileList,
    group: "home",
  });
});
}

export function bindNorthProffInputChange(context) {
  const {elements, state, actions} = context;
elements.northProffInput.addEventListener("change", () => {
  actions.addPendingNorthFiles({
    input: elements.northProffInput,
    names: elements.northProffNames,
    placeholder: "Выберите PROFF-бланки городов",
    files: state.addedNorthProffFiles,
    list: elements.northProffFileList,
    group: "proff",
  });
});
}

export function bindNorthSourceFileChange(context) {
  const {elements, state, actions} = context;
elements.northSourceFile.addEventListener("change", async () => {
  const file = elements.northSourceFile.files[0] || null;
  elements.northSourceName.textContent = file?.name || "Для учета остатков и в пути .xlsx, .xlsm или .xls";
  actions.resetNorthCalculationState();
  if (!file) return;

  elements.northStatus.textContent = "Проверяю таблицу Тюмени...";
  try {
    const workbook = await actions.loadWorkbook(file);
    validateNorthTyumenSourceWorkbook(workbook, file.name);
    elements.northStatus.textContent = "Таблица Тюмени загружена";
  } catch (error) {
    elements.northSourceFile.value = "";
    elements.northSourceName.textContent = "Для учета остатков и в пути .xlsx, .xlsm или .xls";
    elements.northStatus.textContent = "Ошибка";
    alert(error.message || "В это поле можно загрузить только заполненную таблицу Тюмени.");
  }
});
}

export function bindNorthFormSubmit(context) {
  const {elements, state, actions} = context;
elements.northForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const entries = actions.northFilesForMerge();
  if (!entries.length) {
    alert("Добавьте хотя бы один бланк города.");
    return;
  }

  let priceSettings;
  try {priceSettings=state.northPricing.read();}catch(e){alert(e.message);return;}

  elements.northSubmitButton.disabled = true;
  elements.northStatus.textContent = "Проверяю бланки...";
  actions.clearNorthDownloadLinks();
  elements.northResult.classList.add("hidden");
  state.currentNorthResult = null;

  try {
    const blanks = await Promise.all(entries.map(async (entry) => ({
      fileName: entry.file.name,
      workbook: await actions.loadWorkbook(entry.file),
      variant: entry.variant || "",
      variantLabel: entry.variantLabel || "",
    })));
    const tyumenSourceFile = elements.northSourceFile.files[0] || null;
    let tyumenSourceWorkbook = tyumenSourceFile ? await actions.loadWorkbook(tyumenSourceFile) : null;
    if (elements.northTwoSources.checked) actions.assertDifferentTyumenFiles(tyumenSourceFile, elements.northWarehouseFile.files[0]);
    if (elements.northTwoSources.checked) tyumenSourceWorkbook = mergeTyumenSources({
      officeWorkbook: tyumenSourceWorkbook,
      warehouseWorkbook: await actions.loadWorkbook(elements.northWarehouseFile.files[0]),
      officeFileName: tyumenSourceFile.name,
      warehouseFileName: elements.northWarehouseFile.files[0].name,
      brand: actions.selectedNorthBrand(),
    });
    const result = buildNorthOrderFiles(blanks, {
      brand: actions.selectedNorthBrand(),
      tyumenSourceWorkbook,
      tyumenSourceFileName: tyumenSourceFile?.name || "",
    });
    if (!(await actions.confirmNorthMerge(entries, result))) {
      elements.northStatus.textContent = "Готово";
      return;
    }
    state.currentNorthResult = result;
    result.priceSettings=priceSettings;
    actions.renderNorthPlan(result);
    result.budgetPricing=priceOrderRows(actions.liveBudgetRows(true),priceSettings);
    actions.refreshBudgetTotals();
    const supplierRows = result.planRows.filter((row) => Number(row.supplierNeed || 0) > 0).length;
    const tyumenCovered = result.planRows.filter((row) => Number(row.fromTyumen || 0) > 0).length;
    elements.northSummary.textContent = `Города: ${result.uploadedCities.join(", ")}. Позиций к заказу у поставщика: ${supplierRows}. Позиций закрыто остатком Тюмени: ${tyumenCovered}. Перемещений: ${result.transfers.length}.${result.hasTyumenSource ? "" : " Таблица Тюмени не загружена, остаток Тюмени не учитывался."}`;
    elements.northResult.classList.remove("hidden");
    elements.northDownloadButton.disabled = false;
    elements.northStatus.textContent = "Проверьте расчет";
  } catch (error) {
    elements.northStatus.textContent = "Ошибка";
    alert(error.message || "Не удалось соединить бланки.");
  } finally {
    elements.northSubmitButton.disabled = false;
  }
});
}

export function bindNorthPlanBodyInput(context) {
  const {elements, state, actions} = context;
elements.northPlanBody.addEventListener("input", (event) => {
  if (event.target.matches('[data-north-budget-lock]')) {
    event.target.checked ? state.northBudgetLocks.add(event.target.dataset.northBudgetLock) : state.northBudgetLocks.delete(event.target.dataset.northBudgetLock);
    return;
  }
  if (!event.target.matches(".north-actual-input, .north-city-input")) return;
  const rowEl = event.target.closest("tr");
  if (!rowEl) return;
  if (event.target.matches(".north-actual-input")) {
    if (event.target.value.trim() !== '' && Number(event.target.value) === 0) state.northBudgetLocks.add(rowEl.dataset.key);
    rowEl.querySelector('[data-north-budget-lock]').checked = state.northBudgetLocks.has(rowEl.dataset.key);
    event.target.dataset.manual = "true";
    state.northPlanEdits.set(rowEl.dataset.key, event.target.value);
    actions.updateNorthRowDisplay(rowEl, false);
    return;
  }
  actions.updateNorthRowDisplay(rowEl, true);
});
}

export function bindNorthDownloadButtonClick(context) {
  const {elements, state, actions} = context;
elements.northDownloadButton.addEventListener("click", async () => {
  if (!state.currentNorthResult) {
    alert("Сначала соедините бланки.");
    return;
  }

  elements.northDownloadButton.disabled = true;
  elements.northStatus.textContent = "Готовлю файлы...";
  actions.clearNorthDownloadLinks();

  try {
    const edits = actions.collectNorthPlanEdits();
    const shortWarnings = actions.northShortOrderWarnings();
    if (shortWarnings.length && !(await actions.confirmNorthShortOrders(shortWarnings))) {
      elements.northStatus.textContent = "Готово";
      return;
    }
    state.currentNorthResult.budgetPricing=priceOrderRows(actions.liveBudgetRows(true),state.currentNorthResult.priceSettings);
    const finalized = finalizeNorthOrderFiles(state.currentNorthResult, edits, { allowShortSupplierOrder: true });
    const summaryFiles = finalized.summaryFiles?.length
      ? finalized.summaryFiles
      : [{ label: "общий бланк", fileName: finalized.summaryFileName, workbook: finalized.summaryWorkbook }];
    const outputFiles = summaryFiles.map((file) => ({
      label: `Скачать ${file.label || "общий бланк"}`,
      name: file.fileName,
      blob: new Blob([saveXlsx(file.workbook)], {
        type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      }),
    }));
    for (const transfer of finalized.transfers) {
      outputFiles.push({
        label: `Скачать перемещение ${transfer.city.label}`,
        name: transfer.fileName,
        blob: new Blob([await actions.transferWorkbookBytes(transfer)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      });
    }
    if (finalized.orderTable?.rows?.length) {
      outputFiles.push({
        label: "Скачать таблицу заказа",
        name: finalized.orderTable.fileName,
        blob: new Blob([await actions.northOrderTableBytes(finalized.orderTable)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      });
    }

    elements.northSummary.textContent = `Города: ${finalized.uploadedCities.join(", ")}. В общем бланке позиций с заказом: ${finalized.totalsCount}. Перемещений: ${finalized.transfers.length}.${finalized.appendedToSummary.length ? ` Добавлено в конец общего бланка: ${finalized.appendedToSummary.length}.` : ""}${finalized.adjustedToMinimum ? ` Факт отличается от потребности: ${finalized.adjustedToMinimum}.` : ""}`;
    actions.prepareNorthDownloadLinks(outputFiles);
    elements.northStatus.textContent = "Файлы готовы";
  } catch (error) {
    elements.northStatus.textContent = "Ошибка";
    alert(error.message || "Не удалось подготовить файлы.");
  } finally {
    elements.northDownloadButton.disabled = false;
  }
});
}
