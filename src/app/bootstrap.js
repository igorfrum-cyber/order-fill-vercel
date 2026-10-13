import { installTableRecalculation } from "./../ui/recalculation/controller.js";
import { bindDocumentKeydown } from './../ui/order/events.js';
import { bindTwoTyumenSourcesChange } from './../ui/order/events.js';
import { bindNorthTwoSourcesChange } from './../ui/north/events.js';
import { bindNorthFileInputChange } from './../ui/north/events.js';
import { bindNorthHomeInputChange } from './../ui/north/events.js';
import { bindNorthProffInputChange } from './../ui/north/events.js';
import { bindNorthSourceFileChange } from './../ui/north/events.js';
import { bindFormSubmit } from './../ui/order/events.js';
import { bindWarehouseFormSubmit } from './../ui/tyumen/events.js';
import { bindWarehouseRowsInput } from './../ui/tyumen/events.js';
import { bindWarehouseDownloadClick } from './../ui/tyumen/events.js';
import { bindDownloadButtonClick } from './../ui/order/events.js';
import { bindNorthFormSubmit } from './../ui/north/events.js';
import { bindNorthPlanBodyInput } from './../ui/north/events.js';
import { bindNorthDownloadButtonClick } from './../ui/north/events.js';
import { bindMetricsElClick } from './../ui/order/events.js';
import { bindClearFilterButtonClick } from './../ui/order/events.js';
import { createBudgetController } from './../ui/budget/controller.js';
import { createAppLifecycle } from './lifecycle.js';
import { createNorthUploads } from './../ui/north/uploads.js';
import { createOrderReport } from './../ui/order/report.js';
import { createOrderIssues } from './../ui/order/issues.js';
import { createFilesLoader } from './../ui/files/loader.js';
import { createOrderEdits } from './../ui/order/edits.js';
import { createDownloadsBrowser } from './../ui/downloads/browser.js';
import { todayRu } from './../infrastructure/excel/transfer-tables.js';
import { transferWorkbookBytes } from './../infrastructure/excel/transfer-tables.js';
import { northOrderTableBytes } from './../infrastructure/excel/transfer-tables.js';
import { createNorthPlan } from './../ui/north/plan.js';

export function startApplication() {
const elements = {}, state = {}, actions = {todayRu: todayRu, transferWorkbookBytes: transferWorkbookBytes, northOrderTableBytes: northOrderTableBytes};
const context = {elements, state, actions};
Object.assign(actions, createBudgetController(context));
Object.assign(actions, createAppLifecycle(context));
Object.assign(actions, createNorthUploads(context));
Object.assign(actions, createOrderReport(context));
Object.assign(actions, createOrderIssues(context));
Object.assign(actions, createFilesLoader(context));
Object.assign(actions, createOrderEdits(context));
Object.assign(actions, createDownloadsBrowser(context));
Object.assign(actions, createNorthPlan(context));
state.budgetLocks = new Set();

state.northBudgetLocks = new Set();

elements.budgetUndo = document.createElement('button');

elements.northBudgetUndo = document.createElement('button');

state.orderPricing = undefined;

state.northPricing = undefined;

state.budgetPanels = new Map();

window.addEventListener('DOMContentLoaded', actions.installBudgetControls, {once:true});

elements.form = document.querySelector("#uploadForm");

elements.statusEl = document.querySelector("#status");

elements.brandSelect = document.querySelector("#brandSelect");

elements.orderMonth = document.querySelector("#orderMonth");

elements.sourceFile = document.querySelector("#sourceFile");

elements.twoTyumenSources = document.querySelector("#twoTyumenSources");

elements.warehouseSourceFile = document.querySelector("#warehouseSourceFile");

elements.northTwoSources = document.querySelector("#northTwoSources");

elements.northWarehouseFile = document.querySelector("#northWarehouseFile");

elements.warehouseSection = document.querySelector("#warehouseSection");

elements.warehouseModeButton = document.querySelector("#warehouseModeButton");

elements.blankFile = document.querySelector("#blankFile");

elements.homeFile = document.querySelector("#homeFile");

elements.proffFile = document.querySelector("#proffFile");

elements.sourceName = document.querySelector("#sourceName");

elements.blankName = document.querySelector("#blankName");

elements.homeName = document.querySelector("#homeName");

elements.proffName = document.querySelector("#proffName");

elements.blankField = document.querySelector("#blankField");

elements.homeField = document.querySelector("#homeField");

elements.proffField = document.querySelector("#proffField");

elements.resultEl = document.querySelector("#result");

elements.metricsEl = document.querySelector("#metrics");

elements.periodNote = document.querySelector("#periodNote");

elements.reportBody = document.querySelector("#reportBody");

elements.priorityBody = document.querySelector("#priorityBody");

elements.prioritySection = document.querySelector("#prioritySection");

elements.reportTitle = document.querySelector("#reportTitle");

elements.reportSearch = document.querySelector("#reportSearch");

elements.clearFilterButton = document.querySelector("#clearFilterButton");

elements.adjustmentHeader = document.querySelector("#adjustmentHeader");

elements.priorityAdjustmentHeader = document.querySelector("#priorityAdjustmentHeader");

elements.issueReportButton = document.querySelector("#issueReportButton");

elements.downloadButton = document.querySelector("#downloadButton");

elements.includeBlankStocks = document.querySelector("#includeBlankStocks");

elements.downloadLinks = document.querySelector("#downloadLinks");

elements.submitButton = elements.form.querySelector("button");

elements.orderSection = document.querySelector("#orderSection");

elements.northSection = document.querySelector("#northSection");

elements.orderModeButton = document.querySelector("#orderModeButton");

elements.northModeButton = document.querySelector("#northModeButton");

elements.northBrandSelect = document.querySelector("#northBrandSelect");

elements.northForm = document.querySelector("#northForm");

elements.northDefaultUpload = document.querySelector("#northDefaultUpload");

elements.northChristinaUpload = document.querySelector("#northChristinaUpload");

elements.northFileInput = document.querySelector("#northFileInput");

elements.northFileList = document.querySelector("#northFileList");

elements.northHomeInput = document.querySelector("#northHomeInput");

elements.northHomeNames = document.querySelector("#northHomeNames");

elements.northHomeFileList = document.querySelector("#northHomeFileList");

elements.northProffInput = document.querySelector("#northProffInput");

elements.northProffNames = document.querySelector("#northProffNames");

elements.northProffFileList = document.querySelector("#northProffFileList");

elements.northSourceFile = document.querySelector("#northSourceFile");

elements.northNames = document.querySelector("#northNames");

elements.northSourceName = document.querySelector("#northSourceName");

elements.northStatus = document.querySelector("#northStatus");

elements.northResult = document.querySelector("#northResult");

elements.northSummary = document.querySelector("#northSummary");

elements.northDownloadLinks = document.querySelector("#northDownloadLinks");

elements.northPlanBody = document.querySelector("#northPlanBody");

elements.northDownloadButton = document.querySelector("#northDownloadButton");

elements.northSubmitButton = elements.northForm.querySelector("button");

elements.northBackButton = document.querySelector("#northBackButton");

elements.northShortOrderModal = document.querySelector("#northShortOrderModal");

elements.northShortOrderText = document.querySelector("#northShortOrderText");

elements.northShortOrderBack = document.querySelector("#northShortOrderBack");

elements.northShortOrderContinue = document.querySelector("#northShortOrderContinue");

elements.northMergeModal = document.querySelector("#northMergeModal");

elements.northMergeList = document.querySelector("#northMergeList");

elements.northMergeBack = document.querySelector("#northMergeBack");

elements.northMergeContinue = document.querySelector("#northMergeContinue");

state.currentResults = [];

state.currentReportRows = [];

state.currentBlankWorkbooks = new Map();

state.currentSourceWorkbook = null;

state.currentBlankOutputNames = new Map();

state.currentSourceOutputName = "order заполненная таблица.xlsx";

state.currentDownloadUrls = [];

state.currentNorthDownloadUrls = [];

state.currentNorthResult = null;

state.addedNorthFiles = [];

state.addedNorthHomeFiles = [];

state.addedNorthProffFiles = [];

state.northPlanEdits = new Map();

state.isFormFilled = false;

state.activeFilter = null;

state.editState = new Map();

actions.setDefaultOrderMonth();

elements.orderModeButton.addEventListener("click", () => actions.setActiveMode("order"));

elements.northModeButton.addEventListener("click", () => actions.setActiveMode("north"));

elements.warehouseModeButton.addEventListener("click", () => actions.setActiveMode("warehouse"));

elements.northBackButton.addEventListener("click", () => actions.setActiveMode("order"));

actions.setActiveMode("order");

installTableRecalculation({loadWorkbook: actions.loadWorkbook,activate:actions.setActiveMode,brands:elements.brandSelect.innerHTML,month:elements.orderMonth.value});

bindDocumentKeydown(context);

actions.bindFileName(elements.sourceFile, elements.sourceName);

actions.bindFileName(elements.warehouseSourceFile, document.querySelector("#warehouseSourceName"));

bindTwoTyumenSourcesChange(context);

bindNorthTwoSourcesChange(context);

elements.northWarehouseFile.addEventListener("change", () => {
  document.querySelector("#northWarehouseName").textContent = elements.northWarehouseFile.files[0]?.name || ".xlsx, .xlsm или .xls";
  actions.resetNorthCalculationState();
});

actions.bindFileName(elements.blankFile, elements.blankName, ".xlsx, .xlsm или .xls");

actions.bindFileName(elements.homeFile, elements.homeName, ".xlsx или .xlsm");

actions.bindFileName(elements.proffFile, elements.proffName, ".xlsx или .xlsm");

bindNorthFileInputChange(context);

bindNorthHomeInputChange(context);

bindNorthProffInputChange(context);

elements.northFileList.addEventListener("click", actions.handleNorthFileListClick);

elements.northHomeFileList.addEventListener("click", actions.handleNorthFileListClick);

elements.northProffFileList.addEventListener("click", actions.handleNorthFileListClick);

bindNorthSourceFileChange(context);

elements.brandSelect.addEventListener("change", actions.configureBrandFields);

elements.northBrandSelect.addEventListener("change", () => {
  actions.resetNorthUploadedFiles();
  actions.configureNorthBrandUploads();
});

elements.orderMonth.addEventListener("change", actions.resetFillState);

actions.configureBrandFields();

actions.configureNorthBrandUploads();

bindFormSubmit(context);

elements.warehouseForm = document.querySelector("#warehouseForm");

elements.warehouseBrand = document.querySelector("#warehouseBrand");

elements.warehouseBrand.innerHTML = elements.brandSelect.innerHTML;

elements.warehouseResult = document.querySelector("#warehouseResult");

elements.warehouseRows = document.querySelector("#warehouseRows");

elements.warehouseDownload = document.querySelector("#warehouseDownload");

state.warehousePlan = [];

state.warehouseDownloadUrl = null;

state.warehouseRevision = 0;

for (const [inputId, labelId] of [["officeTransferFile", "officeTransferName"], ["warehouseTransferFile", "warehouseTransferName"]]) {
  document.querySelector(`#${inputId}`).addEventListener("change", (event) => {
    document.querySelector(`#${labelId}`).textContent = event.target.files[0]?.name || "Таблица не выбрана";
    state.warehouseRevision += 1;
    state.warehousePlan = [];
    elements.warehouseResult.classList.add("hidden");
  });
}

elements.warehouseBrand.addEventListener("change", () => {
  state.warehouseRevision += 1;
  state.warehousePlan = [];
  elements.warehouseResult.classList.add("hidden");
});

bindWarehouseFormSubmit(context);

bindWarehouseRowsInput(context);

bindWarehouseDownloadClick(context);

bindDownloadButtonClick(context);

elements.issueReportButton.addEventListener("click", actions.downloadIssueReport);

bindNorthFormSubmit(context);

bindNorthPlanBodyInput(context);

bindNorthDownloadButtonClick(context);

elements.reportBody.addEventListener("input", actions.handleReportInput);

elements.priorityBody.addEventListener("input", actions.handleReportInput);

elements.reportBody.addEventListener('input',actions.refreshBudgetTotals);

elements.priorityBody.addEventListener('input',actions.refreshBudgetTotals);

elements.northPlanBody.addEventListener('input',actions.refreshBudgetTotals);

bindMetricsElClick(context);

elements.reportSearch.addEventListener("input", actions.renderReportView);

bindClearFilterButtonClick(context);
}
