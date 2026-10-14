

export function createAppLifecycle(context) {
  const {elements, state, actions} = context;
function setDefaultOrderMonth() {
  const date = new Date();
  date.setMonth(date.getMonth() + 1);
  elements.orderMonth.value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

function setActiveMode(mode) {
  document.querySelector('#recalculateSection')?.classList.toggle('hidden',mode!=='recalculate');
  document.querySelector('#recalculateModeButton')?.classList.toggle('active',mode==='recalculate');
  const isNorth = mode === "north";
  const isWarehouse = mode === "warehouse";
  elements.orderSection.classList.toggle("hidden", mode !== "order");
  elements.warehouseSection.classList.toggle("hidden", !isWarehouse);
  elements.warehouseModeButton.classList.toggle("active", isWarehouse);
  elements.warehouseModeButton.setAttribute("aria-pressed", String(isWarehouse));
  elements.northSection.classList.toggle("hidden", !isNorth);
  elements.orderModeButton.classList.toggle("active", mode === "order");
  elements.northModeButton.classList.toggle("active", isNorth);
  elements.orderModeButton.setAttribute("aria-pressed", String(mode === "order"));
  elements.northModeButton.setAttribute("aria-pressed", String(isNorth));
}

function scrollTargetForKeyboard(element) {
  return element?.closest?.(".table-wrap, .priority-wrap") || document.scrollingElement || document.documentElement;
}

function bindFileName(input, output, placeholder = ".xlsx, .xlsm или .xls") {
  input.addEventListener("change", () => {
    output.textContent = input.files[0]?.name || placeholder;
    resetFillState();
  });
}

function selectedBrand() {
  return elements.brandSelect.value || "angiopharm";
}

function adjustmentLabelForBrand(brand) {
  if (brand === "christina") return "Кратность";
  if (brand === "levissime") return "Кол-во в уп.";
  if (brand === "sothys") return "Округление";
  if (brand === "novacutan") return "Мин. заказ";
  if (brand === "skin_synergy") return "Округление";
  if (brand === "klapp") return "Кратность";
  return "Шт. в коробке";
}

function mainBlankLabelForBrand(brand) {
  if (brand === "levissime") return "LeviSsime";
  if (brand === "sothys") return "SOTHYS";
  if (brand === "novacutan") return "NOVACUTAN";
  if (brand === "skin_synergy") return "Skin Synergy";
  if (brand === "klapp") return "KLAPP";
  return "ANGIO";
}

function selectedNorthBrand() {
  return elements.northBrandSelect.value || "angiopharm";
}

function configureNorthBrandUploads() {
  const isChristina = actions.isNorthChristinaMode();
  elements.northDefaultUpload.classList.toggle("hidden", isChristina);
  elements.northChristinaUpload.classList.toggle("hidden", !isChristina);
  actions.resetNorthCalculationState();
}

function configureBrandFields() {
  const brand = selectedBrand();
  const isChristina = brand === "christina";
  elements.blankField.classList.toggle("hidden", isChristina);
  elements.homeField.classList.toggle("hidden", !isChristina);
  elements.proffField.classList.toggle("hidden", !isChristina);
  elements.blankFile.required = !isChristina;
  elements.homeFile.required = isChristina;
  elements.proffFile.required = isChristina;
  elements.adjustmentHeader.textContent = adjustmentLabelForBrand(brand);
  elements.priorityAdjustmentHeader.textContent = adjustmentLabelForBrand(brand);
  resetFillState();
}

function setSubmitButtonState(state) {
  elements.submitButton.classList.toggle("completed", state === "completed");

  if (state === "processing") {
    elements.submitButton.disabled = true;
    elements.submitButton.innerHTML = "<span>✓</span> Заполняю...";
    return;
  }

  if (state === "completed") {
    elements.submitButton.disabled = true;
    elements.submitButton.innerHTML = "<span>✓</span> Бланк заполнен";
    return;
  }

  elements.submitButton.disabled = false;
  elements.submitButton.innerHTML = "<span>✓</span> Заполнить бланк";
}

function resetFillState() {
  if (!state.isFormFilled && !state.currentResults.length && elements.resultEl.classList.contains("hidden")) {
    setSubmitButtonState("ready");
    return;
  }

  state.isFormFilled = false;
  state.currentResults = [];
  state.currentReportRows = [];
  state.currentBlankWorkbooks = new Map();
  state.currentBlankOutputNames = new Map();
  state.currentSourceWorkbook = null;
  state.activeFilter = null;
  state.editState = new Map();
  state.budgetLocks.clear();
  elements.budgetUndo.hidden = true;
  elements.reportSearch.value = "";
  elements.resultEl.classList.add("hidden");
  elements.downloadButton.disabled = true;
  elements.issueReportButton.disabled = true;
  actions.clearDownloadLinks();
  elements.statusEl.textContent = "Готов к загрузке";
  setSubmitButtonState("ready");
}
  return { setDefaultOrderMonth, setActiveMode, bindFileName, selectedBrand, adjustmentLabelForBrand, mainBlankLabelForBrand, selectedNorthBrand, configureNorthBrandUploads, configureBrandFields, setSubmitButtonState, resetFillState, scrollTargetForKeyboard };
}
