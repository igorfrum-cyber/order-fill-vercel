import { NORTH_ALLOCATION_ORDER, NORTH_CITIES, NORTH_TRANSFER_DISPLAY_ORDER } from "./display-cities.js";
import { defaultNorthActualSupplierOrder } from "./../../domain/north/allocation.js";
import { northTyumenFreeStock } from "./../../domain/warehouse/stock.js";

export function createNorthPlan(context) {
  const {elements, state, actions} = context;
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
        <span>${actions.escapeHtml(city.label)}</span>
        <input class="north-city-input" type="number" min="0" step="1" data-city="${actions.escapeHtml(city.key)}" value="${actions.escapeHtml(String(quantities.get(city.key) || ""))}" />
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
  state.northBudgetLocks.clear();
  elements.northBudgetUndo.hidden = true;
  state.northPlanEdits = new Map();
  elements.northPlanBody.innerHTML = result.planRows
    .map((row) => {
      const value = row.actualSupplierOrder == null ? "" : row.actualSupplierOrder;
      state.northPlanEdits.set(row.key, value);
      const sourceMark = result.hasTyumenSource && !row.hasTyumenSource ? `<div class="north-warning">нет строки в таблице Тюмени</div>` : "";
      return `
        <tr data-key="${actions.escapeHtml(row.key)}">
          <td>${actions.escapeHtml(row.name)}${sourceMark}</td>
          <td class="north-city-cell">${northCityInputs(row)}</td>
          <td>${actions.escapeHtml(formatNorthQuantity(row.northNeed))}</td>
          <td>${actions.escapeHtml(northStockText(row))}</td>
          <td data-role="tyumen-free">${actions.escapeHtml(formatNorthQuantity(row.tyumenFree))}</td>
          <td data-role="from-tyumen">${actions.escapeHtml(formatNorthQuantity(row.fromTyumen))}</td>
          <td data-role="supplier-need">${actions.escapeHtml(formatNorthQuantity(row.supplierNeed))}</td>
          <td>
            <input class="north-actual-input" type="number" min="0" step="1" data-key="${actions.escapeHtml(row.key)}" value="${actions.escapeHtml(String(value))}" data-manual="false" />
            <label class="budget-row-lock"><input type="checkbox" data-north-budget-lock="${actions.escapeHtml(row.key)}"> Закрепить</label>
          </td>
          <td class="north-comment">${actions.escapeHtml(northPlanComment(row, value))}</td>
        </tr>
      `;
    })
    .join("");
}

function defaultNorthActual(row, supplierNeed) {
  return defaultNorthActualSupplierOrder(state.currentNorthResult?.summary || {},row,supplierNeed) ?? '';
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
  const source = state.currentNorthResult?.planRows.find((item) => item.key === rowEl.dataset.key);
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
  for (const rowEl of elements.northPlanBody.querySelectorAll("tr[data-key]")) {
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
  for (const rowEl of elements.northPlanBody.querySelectorAll("tr[data-key]")) {
    const source = state.currentNorthResult?.planRows.find((item) => item.key === rowEl.dataset.key);
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
  elements.northShortOrderText.textContent = `По этим позициям факт у поставщика меньше общей нехватки Тюмени и северных городов.\n\n${preview}${extra}\n\nМожно вернуться и поправить цифры или продолжить скачивание как есть.`;
  elements.northShortOrderModal.classList.remove("hidden");

  return new Promise((resolve) => {
    const close = (value) => {
      elements.northShortOrderModal.classList.add("hidden");
      elements.northShortOrderBack.removeEventListener("click", onBack);
      elements.northShortOrderContinue.removeEventListener("click", onContinue);
      elements.northShortOrderModal.removeEventListener("click", onOverlay);
      document.removeEventListener("keydown", onKeyDown);
      resolve(value);
    };
    const onBack = () => close(false);
    const onContinue = () => close(true);
    const onOverlay = (event) => {
      if (event.target === elements.northShortOrderModal) close(false);
    };
    const onKeyDown = (event) => {
      if (event.key === "Escape") close(false);
    };

    elements.northShortOrderBack.addEventListener("click", onBack);
    elements.northShortOrderContinue.addEventListener("click", onContinue);
    elements.northShortOrderModal.addEventListener("click", onOverlay);
    document.addEventListener("keydown", onKeyDown);
    elements.northShortOrderBack.focus();
  });
}

function confirmNorthMerge(entries, result = null) {
  if (!entries.length) return Promise.resolve(false);
  const rows = result?.confirmationGroups?.length
    ? result.confirmationGroups.map((group) => `${group.city.label}: ${group.variants.join(", ")}`)
    : entries.map((entry) => entry.file.name);
  elements.northMergeList.innerHTML = rows
    .map((text) => `<div class="modal-file-item"><span>${actions.escapeHtml(text)}</span></div>`)
    .join("");
  elements.northMergeModal.classList.remove("hidden");

  return new Promise((resolve) => {
    const close = (value) => {
      elements.northMergeModal.classList.add("hidden");
      elements.northMergeBack.removeEventListener("click", onBack);
      elements.northMergeContinue.removeEventListener("click", onContinue);
      elements.northMergeModal.removeEventListener("click", onOverlay);
      document.removeEventListener("keydown", onKeyDown);
      resolve(value);
    };
    const onBack = () => close(false);
    const onContinue = () => close(true);
    const onOverlay = (event) => {
      if (event.target === elements.northMergeModal) close(false);
    };
    const onKeyDown = (event) => {
      if (event.key === "Escape") close(false);
    };

    elements.northMergeBack.addEventListener("click", onBack);
    elements.northMergeContinue.addEventListener("click", onContinue);
    elements.northMergeModal.addEventListener("click", onOverlay);
    document.addEventListener("keydown", onKeyDown);
    elements.northMergeBack.focus();
  });
}
  return { formatNorthQuantity, formatNorthCommentQuantity, supplierUnitsFromPieces, demandPiecesFromSupplierUnits, northTransferParts, northSupplierOrderText, supplierPartsForNorthActual, northPlanComment, nearestNorthMultiple, northCityInputs, northStockText, renderNorthPlan, defaultNorthActual, northCityQuantities, recalculateNorthRow, updateNorthRowDisplay, collectNorthPlanEdits, northShortOrderWarnings, confirmNorthShortOrders, confirmNorthMerge };
}
