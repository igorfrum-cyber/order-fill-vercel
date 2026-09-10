import "./styles.css";
import { openBudgetDialog, budgetChangeComment } from './budgetDialog.js';
import { createOrderPricing, priceOrderRows, money } from './orderPricing.js';
import { installTableRecalculation } from './recalculateTable.js';
import {
  applyFinalEdits,
  buildNorthOrderFiles,
  finalizeNorthOrderFiles,
  fillWorkbook,
  loadXlsx,
  normalizeOrderValue,
  outputFileName,
  saveXlsx,
  sourceOutputFileName,
  validateNorthTyumenSourceWorkbook,
  mergeTyumenSources,
  buildTyumenWarehousePlan,
  northTyumenFreeStock,
  budgetReportRows,
  budgetOrderRules,
  applyBudgetWorkbookPricing,
  budgetNorthWorkbookPricing,
  workbookPriceOptions,
  defaultNorthActualSupplierOrder,
} from "./workbookProcessor.js";

const budgetLocks = new Set();
const northBudgetLocks = new Set();
const budgetUndo = document.createElement('button');
const northBudgetUndo = document.createElement('button');
let orderPricing, northPricing;
const budgetPanels=new Map();

function liveBudgetRows(north) {
  if(!north)return currentResults.flatMap(r=>budgetReportRows(r,selectedBrand())).map(r=>({...r,quantity:Number(editState.get(r.key)?.value||0)}));
  return (currentNorthResult?.planRows||[]).map(r=>{
    const tr=[...northPlanBody.querySelectorAll('tr[data-key]')].find(el=>el.dataset.key===r.key);
    return {...r.budget,key:r.key,name:r.name,group:r.variant||'main',quantity:Number(tr?.querySelector('.north-actual-input').value||0)};
  });
}

function refreshBudgetTotals() {
  for(const north of [false,true]) {
    const panel=budgetPanels.get(north); if(!panel)continue;
    const rows=liveBudgetRows(north);
    const settings=north?currentNorthResult?.priceSettings:currentResults[0]?.priceSettings;
    if(!settings)continue;
    const priced=priceOrderRows(rows,settings);
    const groups=[...new Set(priced.map(r=>r.group))];
    const host=panel.querySelector('[data-budget-totals]');
    for(const g of groups) {
      let section=[...host.children].find(e=>e.dataset.group===g);
      if(!section){section=document.createElement('div');section.dataset.group=g;section.innerHTML='<strong></strong><label>Желаемая сумма, ₽<input type="number" min="0" step="0.01" data-budget-target></label>';host.append(section);}
      const selected=priced.filter(r=>r.group===g);
      section.querySelector('strong').textContent=`${g==='main'?'Текущая сумма':g.toUpperCase()}: ${selected.some(r=>r.quantity>0&&!r.price)?'Не определена цена':money(selected.reduce((s,r)=>s+Math.round(r.quantity*r.price*100)/100,0))+' ₽'}`;
    }
    for(const child of [...host.children])if(!groups.includes(child.dataset.group))child.remove();
  }
}

function budgetComment(previous, row) {
  const note = budgetChangeComment(row);
  // The report uses a single-line input, which strips newline separators.
  return [previous, note].filter(Boolean).join('; ');
}

function installBudgetControls() {
  for (const north of [false, true]) {
    const anchor = north ? northDownloadButton : downloadButton;
    const button = document.createElement('button');
    const undo = north ? northBudgetUndo : budgetUndo;
    button.type = undo.type = 'button';
    button.className = undo.className = 'secondary';
    button.textContent = 'Заказ до суммы';
    undo.textContent = 'Отменить перерасчет суммы';
    undo.hidden = true;
    const panel=document.createElement('section');panel.className='budget-panel';
    panel.innerHTML='<h3>Заказ до суммы</h3><div data-budget-totals></div><div class="budget-actions"></div>';
    (north?northResult:resultEl).querySelector('.result-head').after(panel);
    panel.querySelector('.budget-actions').append(button,undo);budgetPanels.set(north,panel);
    button.onclick = () => {
      if (north && !currentNorthResult) return alert('Сначала соедините бланки.');
      if (!north && !currentResults.length) return alert('Сначала заполните бланк.');
      if (!north && currentResults.some(r => r.summary.sourceCity !== 'Тюмень')) return alert('Заказ до суммы доступен только для Тюмени. Для северных городов используйте раздел «Север».');
      if (north && !currentNorthResult.hasTyumenSource) return alert('Загрузите таблицу Тюмени для учета продаж и остатков.');
      const brand = north ? selectedNorthBrand() : selectedBrand();
      const locks = north ? northBudgetLocks : budgetLocks;
      const rows = north ? currentNorthResult.planRows.map(r => {
        const tr = [...northPlanBody.querySelectorAll('tr[data-key]')].find(el => el.dataset.key === r.key);
        const calculated = recalculateNorthRow(r,northCityQuantities(tr));
        return { ...r.budget, ...budgetOrderRules(brand,r.name,r.novacutanMinimum ?? r.blankBoxSize),
          key:r.key, name:r.name, group:r.variant || 'main', comment:r.budgetComment || '',
          inventoryKey:r.baseKey || r.key,
          stock:r.tyumenStock, transit:r.tyumenInTransit, outbound:calculated.northNeed,
          quantity:Number(tr.querySelector('.north-actual-input').value || 0), locked:locks.has(r.key),
        };
      }) : currentResults.flatMap(result => budgetReportRows(result,brand)).map(r=>({
        ...r, quantity:Number(editState.get(r.key)?.value || 0), comment:editState.get(r.key)?.comment || '', locked:locks.has(r.key),
      }));
      const counts = new Map();
      for (const r of rows) counts.set(r.inventoryKey,(counts.get(r.inventoryKey)||0)+1);
      for (const r of rows) if (counts.get(r.inventoryKey)>1) r.unsafe=true;
      const settings=north?currentNorthResult.priceSettings:currentResults[0].priceSettings;
      const targets=Object.fromEntries([...panel.querySelector('[data-budget-totals]').children].map(el=>[el.dataset.group,el.querySelector('input').value]));
      openBudgetDialog({ rows:priceOrderRows(rows,settings), fixedPricing:true, initialTargets:targets, christina:brand==='christina', apply:(planned,newLocks)=>{
        // Validate export pricing before changing any manager edits.
        if (north) {
          for (const summary of currentNorthResult.summaries?.length ? currentNorthResult.summaries : [currentNorthResult.summary]) budgetNorthWorkbookPricing(summary, planned);
        } else {
          for (const result of currentResults) applyBudgetWorkbookPricing(result.blankWorkbook,result.blankDetection.sheetName,result.blankDetection.headerRow,planned.filter(r=>r.group===result.blankId));
        }
        const pricingOwners = north ? [currentNorthResult] : currentResults;
        const oldPricing = pricingOwners.map(owner=>owner.budgetPricing);
        for (const owner of pricingOwners) owner.budgetPricing = north ? planned : planned.filter(r=>r.group===owner.blankId);
        // Keep the pre-apply edits, not the initial recommendations, for undo.
        const oldLocks = new Set(locks);
        const oldEdits = new Map([...editState].map(([k,v])=>[k,{...v}]));
        const oldNorth = north ? currentNorthResult.planRows.map(r=>({key:r.key,comment:r.budgetComment})) : [];
        const oldNorthValues = north ? collectNorthPlanEdits() : [];
        locks.clear(); for(const key of newLocks) locks.add(key);
        for(const r of planned) {
          if(north) {
            // Supplier quantity only: never rewrite the city inputs here.
            const source=currentNorthResult.planRows.find(p=>p.key===r.key);
            source.budgetComment=budgetComment(source.budgetComment,r);
            const tr=[...northPlanBody.querySelectorAll('tr[data-key]')].find(el=>el.dataset.key===r.key);
            const input=tr.querySelector('.north-actual-input');
            input.value=r.quantity; input.dataset.manual='true';
            tr.querySelector('[data-north-budget-lock]').checked=locks.has(r.key);
            northPlanEdits.set(r.key,r.quantity); updateNorthRowDisplay(tr);
          } else {
            const edit=editState.get(r.key);
            editState.set(r.key,{...edit,value:r.quantity,comment:budgetComment(edit.comment,r)});
          }
        }
        if(north) clearNorthDownloadLinks(); else { clearDownloadLinks(); renderReportView(); }
        undo.hidden=false;
        refreshBudgetTotals();
        undo.onclick=()=>{
          pricingOwners.forEach((owner,i)=>{owner.budgetPricing=oldPricing[i];});
          locks.clear(); for(const key of oldLocks) locks.add(key);
          if(north) {
            for(const old of oldNorth) currentNorthResult.planRows.find(r=>r.key===old.key).budgetComment=old.comment;
            for(const old of oldNorthValues) {
              const tr=[...northPlanBody.querySelectorAll('tr[data-key]')].find(el=>el.dataset.key===old.key);
              tr.querySelector('.north-actual-input').value=old.actualSupplierOrder ?? '';
              tr.querySelector('[data-north-budget-lock]').checked=locks.has(old.key);
              updateNorthRowDisplay(tr);
            }
            clearNorthDownloadLinks();
          } else { editState=oldEdits; renderReportView(); clearDownloadLinks(); }
          undo.hidden=true;
          refreshBudgetTotals();
        };
      }});
    };
  }
  const ordinaryFiles=()=>selectedBrand()==='christina'?[{group:'home',file:homeFile.files[0]},{group:'proff',file:proffFile.files[0]}].filter(e=>e.file):blankFile.files[0]?[{group:'main',file:blankFile.files[0]}]:[];
  orderPricing=createOrderPricing(form,ordinaryFiles,f=>loadWorkbook(f,{allowLegacyXls:selectedBrand()==='novacutan'}),(w,n)=>workbookPriceOptions(w,n,selectedBrand()),resetFillState);
  northPricing=createOrderPricing(northForm,()=>northFilesForMerge().map(e=>({...e,group:e.variant||'main'})),loadWorkbook,(w,n)=>workbookPriceOptions(w,n,selectedNorthBrand()),()=>{northResult.classList.add('hidden');currentNorthResult=null;clearNorthDownloadLinks();});
  for(const input of [blankFile,homeFile,proffFile,brandSelect])input.addEventListener('change',()=>orderPricing.refresh());
  for(const input of [northFileInput,northHomeInput,northProffInput,northBrandSelect])input.addEventListener('change',()=>northPricing.refresh());
  for(const list of [northFileList,northHomeFileList,northProffFileList])list.addEventListener('click',()=>northPricing.refresh());
}

window.addEventListener('DOMContentLoaded', installBudgetControls, {once:true});

const form = document.querySelector("#uploadForm");
const statusEl = document.querySelector("#status");
const brandSelect = document.querySelector("#brandSelect");
const orderMonth = document.querySelector("#orderMonth");
const sourceFile = document.querySelector("#sourceFile");
const twoTyumenSources = document.querySelector("#twoTyumenSources");
const warehouseSourceFile = document.querySelector("#warehouseSourceFile");
const northTwoSources = document.querySelector("#northTwoSources");
const northWarehouseFile = document.querySelector("#northWarehouseFile");
const warehouseSection = document.querySelector("#warehouseSection");
const warehouseModeButton = document.querySelector("#warehouseModeButton");
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
const northBrandSelect = document.querySelector("#northBrandSelect");
const northForm = document.querySelector("#northForm");
const northDefaultUpload = document.querySelector("#northDefaultUpload");
const northChristinaUpload = document.querySelector("#northChristinaUpload");
const northFileInput = document.querySelector("#northFileInput");
const northFileList = document.querySelector("#northFileList");
const northHomeInput = document.querySelector("#northHomeInput");
const northHomeNames = document.querySelector("#northHomeNames");
const northHomeFileList = document.querySelector("#northHomeFileList");
const northProffInput = document.querySelector("#northProffInput");
const northProffNames = document.querySelector("#northProffNames");
const northProffFileList = document.querySelector("#northProffFileList");
const northSourceFile = document.querySelector("#northSourceFile");
const northNames = document.querySelector("#northNames");
const northSourceName = document.querySelector("#northSourceName");
const northStatus = document.querySelector("#northStatus");
const northResult = document.querySelector("#northResult");
const northSummary = document.querySelector("#northSummary");
const northDownloadLinks = document.querySelector("#northDownloadLinks");
const northPlanBody = document.querySelector("#northPlanBody");
const northDownloadButton = document.querySelector("#northDownloadButton");
const northSubmitButton = northForm.querySelector("button");
const northBackButton = document.querySelector("#northBackButton");
const northShortOrderModal = document.querySelector("#northShortOrderModal");
const northShortOrderText = document.querySelector("#northShortOrderText");
const northShortOrderBack = document.querySelector("#northShortOrderBack");
const northShortOrderContinue = document.querySelector("#northShortOrderContinue");
const northMergeModal = document.querySelector("#northMergeModal");
const northMergeList = document.querySelector("#northMergeList");
const northMergeBack = document.querySelector("#northMergeBack");
const northMergeContinue = document.querySelector("#northMergeContinue");

let currentResults = [];
let currentReportRows = [];
let currentBlankWorkbooks = new Map();
let currentSourceWorkbook = null;
let currentBlankOutputNames = new Map();
let currentSourceOutputName = "order заполненная таблица.xlsx";
let currentDownloadUrls = [];
let currentNorthDownloadUrls = [];
let currentNorthResult = null;
let addedNorthFiles = [];
let addedNorthHomeFiles = [];
let addedNorthProffFiles = [];
let northPlanEdits = new Map();
let isFormFilled = false;
let activeFilter = null;
let editState = new Map();

const NORTH_CITIES = [
  { key: "tyumen", label: "Тюмень" },
  { key: "surgut", label: "Сургут" },
  { key: "nizhnevartovsk", label: "Вартовск" },
  { key: "urengoy", label: "Уренгой" },
];

const NORTH_ALLOCATION_ORDER = ["nizhnevartovsk", "urengoy", "surgut"];
const NORTH_TRANSFER_DISPLAY_ORDER = ["surgut", "nizhnevartovsk", "urengoy"];

function setDefaultOrderMonth() {
  const date = new Date();
  date.setMonth(date.getMonth() + 1);
  orderMonth.value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

setDefaultOrderMonth();

function setActiveMode(mode) {
  document.querySelector('#recalculateSection')?.classList.toggle('hidden',mode!=='recalculate');
  document.querySelector('#recalculateModeButton')?.classList.toggle('active',mode==='recalculate');
  const isNorth = mode === "north";
  const isWarehouse = mode === "warehouse";
  orderSection.classList.toggle("hidden", mode !== "order");
  warehouseSection.classList.toggle("hidden", !isWarehouse);
  warehouseModeButton.classList.toggle("active", isWarehouse);
  warehouseModeButton.setAttribute("aria-pressed", String(isWarehouse));
  northSection.classList.toggle("hidden", !isNorth);
  orderModeButton.classList.toggle("active", mode === "order");
  northModeButton.classList.toggle("active", isNorth);
  orderModeButton.setAttribute("aria-pressed", String(mode === "order"));
  northModeButton.setAttribute("aria-pressed", String(isNorth));
}

orderModeButton.addEventListener("click", () => setActiveMode("order"));
northModeButton.addEventListener("click", () => setActiveMode("north"));
warehouseModeButton.addEventListener("click", () => setActiveMode("warehouse"));
northBackButton.addEventListener("click", () => setActiveMode("order"));
setActiveMode("order");
installTableRecalculation({loadWorkbook,activate:setActiveMode,brands:brandSelect.innerHTML,month:orderMonth.value});

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
bindFileName(warehouseSourceFile, document.querySelector("#warehouseSourceName"));
twoTyumenSources.addEventListener("change", () => {
  document.querySelector("#warehouseSourceField").classList.toggle("hidden", !twoTyumenSources.checked);
  document.querySelector("#tyumenUploadCell").classList.toggle("enabled", twoTyumenSources.checked);
  twoTyumenSources.setAttribute("aria-expanded", String(twoTyumenSources.checked));
  warehouseSourceFile.required = twoTyumenSources.checked;
  warehouseSourceFile.disabled = !twoTyumenSources.checked;
  if (!twoTyumenSources.checked) {
    warehouseSourceFile.value = "";
    document.querySelector("#warehouseSourceName").textContent = ".xlsx, .xlsm или .xls";
  }
  sourceFile.closest("label").querySelector(".label").textContent = twoTyumenSources.checked ? "Офис: Склад Тюмень" : "Таблица заказа товара";
  resetFillState();
});
northTwoSources.addEventListener("change", () => {
  document.querySelector("#northWarehouseField").classList.toggle("hidden", !northTwoSources.checked);
  northWarehouseFile.required = northTwoSources.checked;
  northSourceFile.required = northTwoSources.checked;
  northSourceFile.closest("label").querySelector(".label").textContent = northTwoSources.checked ? "Офис: Склад Тюмень" : "Заполненная таблица Тюмени";
  resetNorthCalculationState();
});
northWarehouseFile.addEventListener("change", () => {
  document.querySelector("#northWarehouseName").textContent = northWarehouseFile.files[0]?.name || ".xlsx, .xlsm или .xls";
  resetNorthCalculationState();
});
bindFileName(blankFile, blankName, ".xlsx, .xlsm или .xls");
bindFileName(homeFile, homeName, ".xlsx или .xlsm");
bindFileName(proffFile, proffName, ".xlsx или .xlsm");

function resetNorthCalculationState() {
  currentNorthResult = null;
  northPlanEdits = new Map();
  clearNorthDownloadLinks();
  northResult.classList.add("hidden");
  northStatus.textContent = "Готов к загрузке";
}

function renderNorthFileList(target, files, group) {
  target.innerHTML = files
    .map((file, index) => `
      <div class="north-file-item">
        <span>${escapeHtml(file.name)}</span>
        <button type="button" data-remove-north-file="${index}" data-north-file-group="${escapeHtml(group)}" aria-label="Удалить ${escapeHtml(file.name)}">×</button>
      </div>
    `)
    .join("");
}

function sameNorthFile(left, right) {
  return left.name === right.name && left.size === right.size && left.lastModified === right.lastModified;
}

function pendingNorthFiles(input) {
  return Array.from(input.files || []);
}

function uniqueNorthFiles(files) {
  const unique = [];
  for (const file of files) {
    if (!unique.some((item) => sameNorthFile(item, file))) unique.push(file);
  }
  return unique;
}

function northFilesForMerge() {
  if (isNorthChristinaMode()) {
    return [
      ...uniqueNorthFiles(addedNorthHomeFiles).map((file) => ({ file, variant: "home", variantLabel: "HOME" })),
      ...uniqueNorthFiles(addedNorthProffFiles).map((file) => ({ file, variant: "proff", variantLabel: "PROFF" })),
    ];
  }
  return uniqueNorthFiles(addedNorthFiles).map((file) => ({ file }));
}

function addPendingNorthFiles({ input, names, placeholder, files, list, group }) {
  const pending = pendingNorthFiles(input);
  if (!pending.length) return;
  let addedCount = 0;
  for (const file of pending) {
    const alreadyAdded = files.some((item) => sameNorthFile(item, file));
    if (alreadyAdded) continue;
    files.push(file);
    addedCount += 1;
  }
  if (!addedCount) {
    alert("Все выбранные бланки уже добавлены.");
    input.value = "";
    names.textContent = placeholder;
    return;
  }
  input.value = "";
  names.textContent = placeholder;
  renderNorthFileList(list, files, group);
  resetNorthCalculationState();
}

northFileInput.addEventListener("change", () => {
  addPendingNorthFiles({
    input: northFileInput,
    names: northNames,
    placeholder: "Выберите один или несколько заполненных бланков",
    files: addedNorthFiles,
    list: northFileList,
    group: "default",
  });
});

northHomeInput.addEventListener("change", () => {
  addPendingNorthFiles({
    input: northHomeInput,
    names: northHomeNames,
    placeholder: "Выберите HOME-бланки городов",
    files: addedNorthHomeFiles,
    list: northHomeFileList,
    group: "home",
  });
});

northProffInput.addEventListener("change", () => {
  addPendingNorthFiles({
    input: northProffInput,
    names: northProffNames,
    placeholder: "Выберите PROFF-бланки городов",
    files: addedNorthProffFiles,
    list: northProffFileList,
    group: "proff",
  });
});

function removeNorthFile(group, index) {
  if (group === "home") {
    addedNorthHomeFiles.splice(index, 1);
    renderNorthFileList(northHomeFileList, addedNorthHomeFiles, "home");
  } else if (group === "proff") {
    addedNorthProffFiles.splice(index, 1);
    renderNorthFileList(northProffFileList, addedNorthProffFiles, "proff");
  } else {
    addedNorthFiles.splice(index, 1);
    renderNorthFileList(northFileList, addedNorthFiles, "default");
  }
  resetNorthCalculationState();
}

function handleNorthFileListClick(event) {
  const button = event.target.closest("[data-remove-north-file]");
  if (!button) return;
  removeNorthFile(button.dataset.northFileGroup || "default", Number(button.dataset.removeNorthFile));
}

northFileList.addEventListener("click", handleNorthFileListClick);
northHomeFileList.addEventListener("click", handleNorthFileListClick);
northProffFileList.addEventListener("click", handleNorthFileListClick);

northSourceFile.addEventListener("change", async () => {
  const file = northSourceFile.files[0] || null;
  northSourceName.textContent = file?.name || "Для учета остатков и в пути .xlsx, .xlsm или .xls";
  resetNorthCalculationState();
  if (!file) return;

  northStatus.textContent = "Проверяю таблицу Тюмени...";
  try {
    const workbook = await loadWorkbook(file);
    validateNorthTyumenSourceWorkbook(workbook, file.name);
    northStatus.textContent = "Таблица Тюмени загружена";
  } catch (error) {
    northSourceFile.value = "";
    northSourceName.textContent = "Для учета остатков и в пути .xlsx, .xlsm или .xls";
    northStatus.textContent = "Ошибка";
    alert(error.message || "В это поле можно загрузить только заполненную таблицу Тюмени.");
  }
});

function selectedBrand() {
  return brandSelect.value || "angiopharm";
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

function isNorthChristinaMode() {
  return selectedNorthBrand() === "christina";
}

function selectedNorthBrand() {
  return northBrandSelect.value || "angiopharm";
}

function configureNorthBrandUploads() {
  const isChristina = isNorthChristinaMode();
  northDefaultUpload.classList.toggle("hidden", isChristina);
  northChristinaUpload.classList.toggle("hidden", !isChristina);
  resetNorthCalculationState();
}

function resetNorthUploadedFiles() {
  addedNorthFiles = [];
  addedNorthHomeFiles = [];
  addedNorthProffFiles = [];
  northFileInput.value = "";
  northHomeInput.value = "";
  northProffInput.value = "";
  northNames.textContent = "Выберите один или несколько заполненных бланков";
  northHomeNames.textContent = "Выберите HOME-бланки городов";
  northProffNames.textContent = "Выберите PROFF-бланки городов";
  renderNorthFileList(northFileList, addedNorthFiles, "default");
  renderNorthFileList(northHomeFileList, addedNorthHomeFiles, "home");
  renderNorthFileList(northProffFileList, addedNorthProffFiles, "proff");
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
northBrandSelect.addEventListener("change", () => {
  resetNorthUploadedFiles();
  configureNorthBrandUploads();
});
orderMonth.addEventListener("change", resetFillState);
configureBrandFields();
configureNorthBrandUploads();

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
  budgetLocks.clear();
  budgetUndo.hidden = true;
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
        <label class="budget-row-lock"><input type="checkbox" data-budget-lock="${escapeHtml(rowKey)}" ${budgetLocks.has(rowKey) ? 'checked' : ''}> Закрепить</label>
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

function assertDifferentTyumenFiles(office, warehouse) {
  if (!office || !warehouse) throw new Error("Загрузите обе таблицы Тюмени: офис и СКЛАД ДОСТАВКА.");
  if (office.name === warehouse.name && office.size === warehouse.size && office.lastModified === warehouse.lastModified) throw new Error("Выбран один и тот же файл для офиса и склада.");
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
  let priceSettings;
  try {priceSettings=orderPricing.read();}catch(e){alert(e.message);return;}

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
    let sourceWorkbook = await loadWorkbook(sourceFile.files[0]);
    if (twoTyumenSources.checked) assertDifferentTyumenFiles(sourceFile.files[0], warehouseSourceFile.files[0]);
    if (twoTyumenSources.checked) sourceWorkbook = mergeTyumenSources({
      officeWorkbook: sourceWorkbook,
      warehouseWorkbook: await loadWorkbook(warehouseSourceFile.files[0]),
      officeFileName: sourceFile.files[0].name,
      warehouseFileName: warehouseSourceFile.files[0].name,
      brand,
    });
    const blankWorkbooks = await Promise.all(blankInputs.map((item) => loadWorkbook(item.file, { allowLegacyXls: brand === "novacutan" })));
    const results = blankInputs.map((item, index) => fillWorkbook({
      sourceWorkbook,
      sourceFileName: sourceFile.files[0].name,
      blankWorkbook: blankWorkbooks[index],
      orderMonth: orderMonth.value,
      brand,
      blankId: item.id,
      blankLabel: item.label,
    }));

    currentResults = results;
    for(const result of results){result.priceSettings=priceSettings;result.budgetPricing=priceOrderRows(budgetReportRows(result,brand),priceSettings);}
    currentSourceWorkbook = sourceWorkbook;
    currentBlankWorkbooks = new Map(results.map((result) => [result.blankId, result.blankWorkbook]));
    for (const [index, item] of blankInputs.entries()) {
      currentBlankWorkbooks.set(item.id, results[index].blankWorkbook);
      currentBlankOutputNames.set(item.id, outputFileName(item.file.name, results[index].summary.sourceCity));
    }
    currentSourceOutputName = twoTyumenSources.checked ? `${brandSelect.selectedOptions[0].textContent} Тюмень общая заполненная таблица.xlsx` : sourceOutputFileName(sourceFile.files[0].name);

    const rows = [...results.flatMap((result) => result.reportRows), ...sourceDuplicateRows(results), ...missingInBlankRows(results)];
    currentReportRows = rows;
    editState = new Map(rows.map((row) => [row.key || `${row.blankId}:${row.blankRow}`, { value: row.inserted ?? "", comment: initialComment(row) }]));
    for (const row of rows) if (row.hasOrderedFact && Number(row.orderedFact) === 0) budgetLocks.add(row.key);
    renderMetrics(combinedSummary(results));
    renderReportView();
    refreshBudgetTotals();
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
    const baseline = baselineForReportRow(rowInfo);
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
  const baseline = baselineForReportRow(rowInfo);
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

const warehouseForm = document.querySelector("#warehouseForm");
const warehouseBrand = document.querySelector("#warehouseBrand");
warehouseBrand.innerHTML = brandSelect.innerHTML;
const warehouseResult = document.querySelector("#warehouseResult");
const warehouseRows = document.querySelector("#warehouseRows");
const warehouseDownload = document.querySelector("#warehouseDownload");
let warehousePlan = [];
let warehouseDownloadUrl = null;
let warehouseRevision = 0;
for (const [inputId, labelId] of [["officeTransferFile", "officeTransferName"], ["warehouseTransferFile", "warehouseTransferName"]]) {
  document.querySelector(`#${inputId}`).addEventListener("change", (event) => {
    document.querySelector(`#${labelId}`).textContent = event.target.files[0]?.name || "Таблица не выбрана";
    warehouseRevision += 1;
    warehousePlan = [];
    warehouseResult.classList.add("hidden");
  });
}
warehouseBrand.addEventListener("change", () => {
  warehouseRevision += 1;
  warehousePlan = [];
  warehouseResult.classList.add("hidden");
});
warehouseForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const button = warehouseForm.querySelector("button");
  const revision = ++warehouseRevision;
  button.disabled = true;
  warehouseResult.classList.add("hidden");
  try {
    const office = document.querySelector("#officeTransferFile").files[0];
    const warehouse = document.querySelector("#warehouseTransferFile").files[0];
    assertDifferentTyumenFiles(office, warehouse);
    const plan = buildTyumenWarehousePlan({
      officeWorkbook: await loadWorkbook(office), warehouseWorkbook: await loadWorkbook(warehouse),
      officeFileName: office.name, warehouseFileName: warehouse.name, brand: warehouseBrand.value,
    });
    if (revision !== warehouseRevision) return;
    warehousePlan = plan;
    warehouseRows.innerHTML = plan.map((row, index) => `<tr><td>${escapeHtml(row.article)}</td><td>${escapeHtml(row.name)}</td><td>${row.officeStock}</td><td>${row.warehouseStock}</td><td>${row.target}</td><td><input class="warehouse-quantity" aria-label="${escapeHtml(`Переместить: ${row.name}`)}" type="number" min="0" max="${row.warehouseStock}" step="1" value="${row.quantity}" data-index="${index}" /></td><td data-office-after>${row.officeStock + row.quantity}</td><td data-warehouse-after>${row.warehouseStock - row.quantity}</td></tr>`).join("");
    warehouseResult.classList.remove("hidden");
    warehouseDownload.disabled = !plan.length;
  } catch (error) {
    warehousePlan = [];
    alert(error.message);
  } finally { button.disabled = false; }
});
warehouseRows.addEventListener("input", (event) => {
  const input = event.target.closest(".warehouse-quantity");
  if (!input) return;
  const row = warehousePlan[Number(input.dataset.index)];
  const value = input.value === "" ? 0 : Number(input.value);
  const valid = Number.isInteger(value) && value >= 0 && value <= row.warehouseStock;
  input.setCustomValidity(valid ? "" : "Количество должно быть целым, неотрицательным и не больше остатка склада.");
  input.closest("tr").querySelector("[data-office-after]").textContent = valid ? row.officeStock + value : "—";
  input.closest("tr").querySelector("[data-warehouse-after]").textContent = valid ? row.warehouseStock - value : "—";
  row.quantity = valid ? value : NaN;
});
warehouseDownload.addEventListener("click", async () => {
  for (const input of warehouseRows.querySelectorAll("input")) if (!input.reportValidity()) return;
  const items = warehousePlan.filter((row) => row.quantity > 0);
  if (!items.length) { alert("Нет позиций для перемещения."); return; }
  warehouseDownload.disabled = true;
  try {
    const bytes = await transferWorkbookBytes({ city: { warehouse: "Склад Тюмень" }, items });
    if (warehouseDownloadUrl) URL.revokeObjectURL(warehouseDownloadUrl);
    warehouseDownloadUrl = URL.createObjectURL(new Blob([bytes], { type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" }));
    triggerDownload(warehouseDownloadUrl, `${warehouseBrand.selectedOptions[0].textContent} Заказ на перемещение Склад Тюмень - Офис Тюмень ${todayRu()}.xlsx`);
  } catch (error) { alert(error.message); }
  finally { warehouseDownload.disabled = false; }
});

function transferWorkbookBytes(transfer) {
  return import("xlsx").then(({ utils, write }) => {
    const rows = [
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", `Заказ на перемещение от ${todayRu()}`, "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "", "", "", "", "", ""],
      ["", "", "Отправитель:", "СКЛАД ДОСТАВКА", "Получатель:", transfer.city.warehouse, "", ""],
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

function northOrderTableBytes(table) {
  return import("xlsx").then(({ utils, write }) => {
    const rows = [
      ["Позиция", "Заказано", "Комментарий"],
      ...table.rows.map((item) => [item.name, item.quantity, item.comment]),
    ];
    const workbook = utils.book_new();
    const sheet = utils.aoa_to_sheet(rows);
    sheet["!cols"] = [
      { wch: 72 },
      { wch: 14 },
      { wch: 80 },
    ];
    utils.book_append_sheet(workbook, sheet, "Заказ");
    return write(workbook, { bookType: "xlsx", type: "array" });
  });
}

function formatNorthQuantity(value) {
  const number = Number(value || 0);
  if (!Number.isFinite(number) || number <= 0) return "";
  if (Number.isInteger(number)) return String(number);
  return String(Number(number.toFixed(2)));
}

function formatNorthCommentQuantity(value) {
  const number = Math.round(Number(value || 0));
  return Number.isFinite(number) && number > 0 ? String(number) : "";
}

function supplierUnitsFromPieces(quantity, unitSize = 1) {
  const number = Number(quantity || 0);
  const size = Number(unitSize || 1);
  if (!Number.isFinite(number) || number <= 0) return null;
  if (!Number.isFinite(size) || size <= 1) return Number(number.toFixed(2));
  return Math.ceil(number / size);
}

function demandPiecesFromSupplierUnits(quantity, unitSize = 1) {
  const number = Number(quantity || 0);
  const size = Number(unitSize || 1);
  if (!Number.isFinite(number) || number <= 0) return 0;
  return Number((number * (Number.isFinite(size) && size > 0 ? size : 1)).toFixed(2));
}

function northTransferParts(row) {
  const quantities = new Map((row.cities || []).map((city) => [city.key, Number(city.quantity || 0)]));
  return NORTH_TRANSFER_DISPLAY_ORDER
    .map((cityKey) => {
      const city = NORTH_CITIES.find((item) => item.key === cityKey);
      return city ? { ...city, quantity: quantities.get(city.key) || 0 } : null;
    })
    .filter(Boolean);
}

function northSupplierOrderText(row, actual) {
  const actualRounded = Math.round(Number(actual || 0));
  const neededRounded = Math.round(Number(row.supplierNeed || 0));
  const extraRounded = Math.max(0, actualRounded - neededRounded);
  const unitNote = Number(row.supplierUnitSize || 1) > 1 ? ` коробок по ${Number(row.supplierUnitSize)}` : "";
  if (extraRounded > 0) {
    return `${neededRounded} + ${extraRounded} (${row.budgetComment ? 'сверх потребности' : 'до минимального'}) = ${actualRounded}${unitNote}`;
  }
  return `${formatNorthQuantity(actual)}${unitNote}`;
}

function supplierPartsForNorthActual(row, actualValue = row.actualSupplierOrder) {
  const actual = demandPiecesFromSupplierUnits(actualValue, row.supplierUnitSize);
  const northParts = (row.supplierParts || []).filter((part) => part.key !== "tyumen");
  const northNeed = northParts.reduce((sum, part) => sum + Number(part.quantity || 0), 0);
  const tyumenQuantity = Math.max(0, actual - northNeed);
  return [
    ...(tyumenQuantity > 0 ? [{ key: "tyumen", label: "Тюмень", quantity: Number(tyumenQuantity.toFixed(2)) }] : []),
    ...northParts,
  ];
}

function northPlanComment(row, actualValue = row.actualSupplierOrder) {
  const lines = [];
  const actual = Number(actualValue || 0);
  const supplierParts = supplierPartsForNorthActual(row, actualValue);
  const tyumenSupplier = supplierParts.find((part) => part.key === "tyumen");
  const transferParts = northTransferParts(row);

  if (actual > 0) lines.push(`Заказать у поставщика: ${northSupplierOrderText(row, actual)}`);
  for (const part of transferParts) {
    if (Number(part.quantity || 0) > 0) lines.push(`Отправить в ${part.label}: ${formatNorthQuantity(part.quantity)}`);
  }
  if (Number(tyumenSupplier?.quantity || 0) > 0) {
    lines.push(`Оставить в Тюмени: ${formatNorthCommentQuantity(tyumenSupplier.quantity)}`);
  }
  if (!lines.length && row.northNeed > 0) lines.push("Закрывается остатком Тюмени");
  if (row.budgetComment) lines.push(row.budgetComment);
  return lines.join("\n");
}

function nearestNorthMultiple(value, multiple) {
  const number = Number(value || 0);
  const step = Math.round(Number(multiple));
  if (!Number.isFinite(number) || number <= 0) return "";
  if (!Number.isFinite(step) || step <= 0) return Number(number.toFixed(2));
  const lower = Math.floor(number / step) * step;
  const upper = Math.ceil(number / step) * step;
  if (lower <= 0) return upper;
  return upper - number <= number - lower ? upper : lower;
}

function northCityInputs(row) {
  const quantities = new Map((row.cities || []).map((city) => [city.key, city.quantity]));
  return NORTH_CITIES
    .filter((city) => quantities.has(city.key))
    .map((city) => `
      <label class="north-city-field">
        <span>${escapeHtml(city.label)}</span>
        <input class="north-city-input" type="number" min="0" step="1" data-city="${escapeHtml(city.key)}" value="${escapeHtml(String(quantities.get(city.key) || ""))}" />
      </label>
    `)
    .join("") || `<span class="muted">нет городских количеств</span>`;
}

function northStockText(row) {
  const parts = [`ост. ${formatNorthQuantity(row.tyumenStock) || "0"}`];
  if (row.tyumenWarehouseStock != null) {
    parts.push(`офис ${formatNorthQuantity(row.tyumenStock - row.tyumenWarehouseStock) || "0"}`);
    parts.push(`склад ${formatNorthQuantity(row.tyumenWarehouseStock) || "0"}`);
  }
  if (Number(row.tyumenInTransit || 0) > 0) parts.push(`в пути ${formatNorthQuantity(row.tyumenInTransit)}`);
  if (Number(row.tyumenTarget || 0) > 0) parts.push(`цель ${formatNorthQuantity(row.tyumenTarget)}`);
  return parts.join(", ");
}

function renderNorthPlan(result) {
  northBudgetLocks.clear();
  northBudgetUndo.hidden = true;
  northPlanEdits = new Map();
  northPlanBody.innerHTML = result.planRows
    .map((row) => {
      const value = row.actualSupplierOrder == null ? "" : row.actualSupplierOrder;
      northPlanEdits.set(row.key, value);
      const sourceMark = result.hasTyumenSource && !row.hasTyumenSource ? `<div class="north-warning">нет строки в таблице Тюмени</div>` : "";
      return `
        <tr data-key="${escapeHtml(row.key)}">
          <td>${escapeHtml(row.name)}${sourceMark}</td>
          <td class="north-city-cell">${northCityInputs(row)}</td>
          <td>${escapeHtml(formatNorthQuantity(row.northNeed))}</td>
          <td>${escapeHtml(northStockText(row))}</td>
          <td data-role="tyumen-free">${escapeHtml(formatNorthQuantity(row.tyumenFree))}</td>
          <td data-role="from-tyumen">${escapeHtml(formatNorthQuantity(row.fromTyumen))}</td>
          <td data-role="supplier-need">${escapeHtml(formatNorthQuantity(row.supplierNeed))}</td>
          <td>
            <input class="north-actual-input" type="number" min="0" step="1" data-key="${escapeHtml(row.key)}" value="${escapeHtml(String(value))}" data-manual="false" />
            <label class="budget-row-lock"><input type="checkbox" data-north-budget-lock="${escapeHtml(row.key)}"> Закрепить</label>
          </td>
          <td class="north-comment">${escapeHtml(northPlanComment(row, value))}</td>
        </tr>
      `;
    })
    .join("");
}

function defaultNorthActual(row, supplierNeed) {
  return defaultNorthActualSupplierOrder(currentNorthResult?.summary || {},row,supplierNeed) ?? '';
}

function northCityQuantities(rowEl) {
  const quantities = {};
  for (const input of rowEl.querySelectorAll(".north-city-input")) {
    quantities[input.dataset.city] = input.value.trim() === "" ? 0 : Number(input.value);
  }
  return quantities;
}

function recalculateNorthRow(row, quantities) {
  const cityMap = new Map(Object.entries(quantities).map(([key, value]) => [key, Number(value || 0)]));
  const tyumenUploadedOrder = Number(cityMap.get("tyumen") || 0);
  const tyumenPlannedOrder = cityMap.has("tyumen") ? tyumenUploadedOrder : Number(row.tyumenPlannedOrder || 0);
  const tyumenSupplierNeed = Math.max(0, tyumenPlannedOrder);
  const supplierUnitSize = Number(row.supplierUnitSize || 1);
  const free = northTyumenFreeStock(Number(row.tyumenStock || 0), Number(row.tyumenInTransit || 0), tyumenPlannedOrder, Number(row.tyumenTarget || 0), row.tyumenWarehouseStock, row.tyumenWarehouseTransit);
  let freeLeft = free;
  const supplierParts = [];
  const tyumenParts = [];
  let northNeed = 0;
  let supplierNorthNeed = 0;
  let fromTyumen = 0;
  const cities = NORTH_CITIES
    .map((city) => ({ key: city.key, label: city.label, quantity: Number((cityMap.get(city.key) || 0).toFixed(2)) }))
    .filter((city) => city.quantity > 0);

  if (tyumenSupplierNeed > 0) supplierParts.push({ key: "tyumen", label: "Тюмень", quantity: Number(tyumenSupplierNeed.toFixed(2)) });

  for (const cityKey of NORTH_ALLOCATION_ORDER) {
    const city = NORTH_CITIES.find((item) => item.key === cityKey);
    if (!city) continue;
    const quantity = Number(cityMap.get(city.key) || 0);
    if (quantity <= 0) continue;
    northNeed += quantity;
    const fromTyumenPart = Math.min(quantity, freeLeft);
    const fromSupplierPart = quantity - fromTyumenPart;
    freeLeft -= fromTyumenPart;
    fromTyumen += fromTyumenPart;
    supplierNorthNeed += fromSupplierPart;
    if (fromTyumenPart > 0) tyumenParts.push({ key: city.key, label: city.label, quantity: Number(fromTyumenPart.toFixed(2)) });
    if (fromSupplierPart > 0) supplierParts.push({ key: city.key, label: city.label, quantity: Number(fromSupplierPart.toFixed(2)) });
  }

  const supplierDemandNeed = Number((tyumenSupplierNeed + supplierNorthNeed).toFixed(2));
  const supplierNeed = supplierUnitsFromPieces(supplierDemandNeed, supplierUnitSize) || 0;
  return {
    ...row,
    northNeed: Number(northNeed.toFixed(2)),
    cities,
    tyumenFree: Number(free.toFixed(2)),
    fromTyumen: Number(fromTyumen.toFixed(2)),
    supplierNorthNeed: Number(supplierNorthNeed.toFixed(2)),
    supplierDemandNeed,
    supplierNeed,
    supplierParts,
    tyumenParts,
  };
}

function updateNorthRowDisplay(rowEl, changedCity = false) {
  const source = currentNorthResult?.planRows.find((item) => item.key === rowEl.dataset.key);
  if (!source) return;
  const calculated = recalculateNorthRow(source, northCityQuantities(rowEl));
  const actualInput = rowEl.querySelector(".north-actual-input");
  if (changedCity && actualInput?.dataset.manual !== "true") {
    actualInput.value = defaultNorthActual(calculated, calculated.supplierNeed);
  }
  const actual = actualInput?.value.trim() === "" ? null : Number(actualInput.value);
  rowEl.querySelector('[data-role="tyumen-free"]').textContent = formatNorthQuantity(calculated.tyumenFree);
  rowEl.querySelector('[data-role="from-tyumen"]').textContent = formatNorthQuantity(calculated.fromTyumen);
  rowEl.querySelector('[data-role="supplier-need"]').textContent = formatNorthQuantity(calculated.supplierNeed);
  rowEl.querySelector(".north-comment").textContent = northPlanComment(calculated, actual);
}

function collectNorthPlanEdits() {
  const edits = [];
  for (const rowEl of northPlanBody.querySelectorAll("tr[data-key]")) {
    const input = rowEl.querySelector(".north-actual-input");
    const value = input.value.trim();
    if (value && Number(value) < 0) throw new Error("Фактический заказ у поставщика не может быть отрицательным.");
    edits.push({
      key: rowEl.dataset.key,
      cities: northCityQuantities(rowEl),
      actualSupplierOrder: value === "" ? null : Number(value),
    });
  }
  return edits;
}

function northShortOrderWarnings() {
  const warnings = [];
  for (const rowEl of northPlanBody.querySelectorAll("tr[data-key]")) {
    const source = currentNorthResult?.planRows.find((item) => item.key === rowEl.dataset.key);
    if (!source) continue;
    const calculated = recalculateNorthRow(source, northCityQuantities(rowEl));
    const input = rowEl.querySelector(".north-actual-input");
    const actual = input?.value.trim() === "" ? 0 : Number(input.value);
    const need = Number(calculated.supplierNeed || 0);
    if (calculated.allowsRoundedShortSupplierOrder && actual === defaultNorthActual(calculated, need)) continue;
    if (Number.isFinite(actual) && actual < need) {
      warnings.push({
        name: calculated.name,
        actual,
        need,
      });
    }
  }
  return warnings;
}

function confirmNorthShortOrders(warnings) {
  if (!warnings.length) return Promise.resolve(true);
  const preview = warnings
    .slice(0, 8)
    .map((item) => `• ${item.name}: факт ${formatNorthQuantity(item.actual) || "0"}, нужно ${formatNorthQuantity(item.need)}`)
    .join("\n");
  const extra = warnings.length > 8 ? `\nЕще позиций: ${warnings.length - 8}` : "";
  northShortOrderText.textContent = `По этим позициям факт у поставщика меньше общей нехватки Тюмени и северных городов.\n\n${preview}${extra}\n\nМожно вернуться и поправить цифры или продолжить скачивание как есть.`;
  northShortOrderModal.classList.remove("hidden");

  return new Promise((resolve) => {
    const close = (value) => {
      northShortOrderModal.classList.add("hidden");
      northShortOrderBack.removeEventListener("click", onBack);
      northShortOrderContinue.removeEventListener("click", onContinue);
      northShortOrderModal.removeEventListener("click", onOverlay);
      document.removeEventListener("keydown", onKeyDown);
      resolve(value);
    };
    const onBack = () => close(false);
    const onContinue = () => close(true);
    const onOverlay = (event) => {
      if (event.target === northShortOrderModal) close(false);
    };
    const onKeyDown = (event) => {
      if (event.key === "Escape") close(false);
    };

    northShortOrderBack.addEventListener("click", onBack);
    northShortOrderContinue.addEventListener("click", onContinue);
    northShortOrderModal.addEventListener("click", onOverlay);
    document.addEventListener("keydown", onKeyDown);
    northShortOrderBack.focus();
  });
}

function confirmNorthMerge(entries, result = null) {
  if (!entries.length) return Promise.resolve(false);
  const rows = result?.confirmationGroups?.length
    ? result.confirmationGroups.map((group) => `${group.city.label}: ${group.variants.join(", ")}`)
    : entries.map((entry) => entry.file.name);
  northMergeList.innerHTML = rows
    .map((text) => `<div class="modal-file-item"><span>${escapeHtml(text)}</span></div>`)
    .join("");
  northMergeModal.classList.remove("hidden");

  return new Promise((resolve) => {
    const close = (value) => {
      northMergeModal.classList.add("hidden");
      northMergeBack.removeEventListener("click", onBack);
      northMergeContinue.removeEventListener("click", onContinue);
      northMergeModal.removeEventListener("click", onOverlay);
      document.removeEventListener("keydown", onKeyDown);
      resolve(value);
    };
    const onBack = () => close(false);
    const onContinue = () => close(true);
    const onOverlay = (event) => {
      if (event.target === northMergeModal) close(false);
    };
    const onKeyDown = (event) => {
      if (event.key === "Escape") close(false);
    };

    northMergeBack.addEventListener("click", onBack);
    northMergeContinue.addEventListener("click", onContinue);
    northMergeModal.addEventListener("click", onOverlay);
    document.addEventListener("keydown", onKeyDown);
    northMergeBack.focus();
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
      result.budgetPricing=priceOrderRows(liveBudgetRows(false).filter(r=>r.group===result.blankId),result.priceSettings);
      files.push({
        label: `Скачать ${result.blankLabel || "бланк"}`,
        name: currentBlankOutputNames.get(result.blankId) || "blank заполненный.xlsx",
        blob: new Blob([saveXlsx(applyBudgetWorkbookPricing(result.blankWorkbook,result.blankDetection.sheetName,result.blankDetection.headerRow,result.budgetPricing))], {
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
  const entries = northFilesForMerge();
  if (!entries.length) {
    alert("Добавьте хотя бы один бланк города.");
    return;
  }

  let priceSettings;
  try {priceSettings=northPricing.read();}catch(e){alert(e.message);return;}

  northSubmitButton.disabled = true;
  northStatus.textContent = "Проверяю бланки...";
  clearNorthDownloadLinks();
  northResult.classList.add("hidden");
  currentNorthResult = null;

  try {
    const blanks = await Promise.all(entries.map(async (entry) => ({
      fileName: entry.file.name,
      workbook: await loadWorkbook(entry.file),
      variant: entry.variant || "",
      variantLabel: entry.variantLabel || "",
    })));
    const tyumenSourceFile = northSourceFile.files[0] || null;
    let tyumenSourceWorkbook = tyumenSourceFile ? await loadWorkbook(tyumenSourceFile) : null;
    if (northTwoSources.checked) assertDifferentTyumenFiles(tyumenSourceFile, northWarehouseFile.files[0]);
    if (northTwoSources.checked) tyumenSourceWorkbook = mergeTyumenSources({
      officeWorkbook: tyumenSourceWorkbook,
      warehouseWorkbook: await loadWorkbook(northWarehouseFile.files[0]),
      officeFileName: tyumenSourceFile.name,
      warehouseFileName: northWarehouseFile.files[0].name,
      brand: selectedNorthBrand(),
    });
    const result = buildNorthOrderFiles(blanks, {
      brand: selectedNorthBrand(),
      tyumenSourceWorkbook,
      tyumenSourceFileName: tyumenSourceFile?.name || "",
    });
    if (!(await confirmNorthMerge(entries, result))) {
      northStatus.textContent = "Готово";
      return;
    }
    currentNorthResult = result;
    result.priceSettings=priceSettings;
    renderNorthPlan(result);
    result.budgetPricing=priceOrderRows(liveBudgetRows(true),priceSettings);
    refreshBudgetTotals();
    const supplierRows = result.planRows.filter((row) => Number(row.supplierNeed || 0) > 0).length;
    const tyumenCovered = result.planRows.filter((row) => Number(row.fromTyumen || 0) > 0).length;
    northSummary.textContent = `Города: ${result.uploadedCities.join(", ")}. Позиций к заказу у поставщика: ${supplierRows}. Позиций закрыто остатком Тюмени: ${tyumenCovered}. Перемещений: ${result.transfers.length}.${result.hasTyumenSource ? "" : " Таблица Тюмени не загружена, остаток Тюмени не учитывался."}`;
    northResult.classList.remove("hidden");
    northDownloadButton.disabled = false;
    northStatus.textContent = "Проверьте расчет";
  } catch (error) {
    northStatus.textContent = "Ошибка";
    alert(error.message || "Не удалось соединить бланки.");
  } finally {
    northSubmitButton.disabled = false;
  }
});

northPlanBody.addEventListener("input", (event) => {
  if (event.target.matches('[data-north-budget-lock]')) {
    event.target.checked ? northBudgetLocks.add(event.target.dataset.northBudgetLock) : northBudgetLocks.delete(event.target.dataset.northBudgetLock);
    return;
  }
  if (!event.target.matches(".north-actual-input, .north-city-input")) return;
  const rowEl = event.target.closest("tr");
  if (!rowEl) return;
  if (event.target.matches(".north-actual-input")) {
    if (event.target.value.trim() !== '' && Number(event.target.value) === 0) northBudgetLocks.add(rowEl.dataset.key);
    rowEl.querySelector('[data-north-budget-lock]').checked = northBudgetLocks.has(rowEl.dataset.key);
    event.target.dataset.manual = "true";
    northPlanEdits.set(rowEl.dataset.key, event.target.value);
    updateNorthRowDisplay(rowEl, false);
    return;
  }
  updateNorthRowDisplay(rowEl, true);
});

northDownloadButton.addEventListener("click", async () => {
  if (!currentNorthResult) {
    alert("Сначала соедините бланки.");
    return;
  }

  northDownloadButton.disabled = true;
  northStatus.textContent = "Готовлю файлы...";
  clearNorthDownloadLinks();

  try {
    const edits = collectNorthPlanEdits();
    const shortWarnings = northShortOrderWarnings();
    if (shortWarnings.length && !(await confirmNorthShortOrders(shortWarnings))) {
      northStatus.textContent = "Готово";
      return;
    }
    currentNorthResult.budgetPricing=priceOrderRows(liveBudgetRows(true),currentNorthResult.priceSettings);
    const finalized = finalizeNorthOrderFiles(currentNorthResult, edits, { allowShortSupplierOrder: true });
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
        blob: new Blob([await transferWorkbookBytes(transfer)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      });
    }
    if (finalized.orderTable?.rows?.length) {
      outputFiles.push({
        label: "Скачать таблицу заказа",
        name: finalized.orderTable.fileName,
        blob: new Blob([await northOrderTableBytes(finalized.orderTable)], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        }),
      });
    }

    northSummary.textContent = `Города: ${finalized.uploadedCities.join(", ")}. В общем бланке позиций с заказом: ${finalized.totalsCount}. Перемещений: ${finalized.transfers.length}.${finalized.appendedToSummary.length ? ` Добавлено в конец общего бланка: ${finalized.appendedToSummary.length}.` : ""}${finalized.adjustedToMinimum ? ` Факт отличается от потребности: ${finalized.adjustedToMinimum}.` : ""}`;
    prepareNorthDownloadLinks(outputFiles);
    northStatus.textContent = "Файлы готовы";
  } catch (error) {
    northStatus.textContent = "Ошибка";
    alert(error.message || "Не удалось подготовить файлы.");
  } finally {
    northDownloadButton.disabled = false;
  }
});

function handleReportInput(event) {
  if (event.target.matches('[data-budget-lock]')) {
    event.target.checked ? budgetLocks.add(event.target.dataset.budgetLock) : budgetLocks.delete(event.target.dataset.budgetLock);
    return;
  }
  if (event.target.matches(".qty-input, .comment-input")) {
    const key = event.target.dataset.key;
    const current = editState.get(key) || { value: "", comment: "" };
    if (event.target.matches(".qty-input")) {
      current.value = event.target.value;
      if (String(current.value).trim() !== '' && Number(current.value) === 0) budgetLocks.add(key);
      event.target.closest('tr').querySelector('[data-budget-lock]').checked = budgetLocks.has(key);
    }
    if (event.target.matches(".comment-input")) current.comment = event.target.value;
    editState.set(key, current);
    const row = event.target.closest("tr");
    row?.classList.remove("invalid");
    if (row) updateCommentHint(row);
  }
}

reportBody.addEventListener("input", handleReportInput);
priorityBody.addEventListener("input", handleReportInput);
reportBody.addEventListener('input',refreshBudgetTotals);
priorityBody.addEventListener('input',refreshBudgetTotals);
northPlanBody.addEventListener('input',refreshBudgetTotals);

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
