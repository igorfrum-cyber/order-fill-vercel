import { buildTyumenWarehousePlan } from "./../../workflows/tyumen/merge.js";

export function bindWarehouseFormSubmit(context) {
  const {elements, state, actions} = context;
elements.warehouseForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const button = elements.warehouseForm.querySelector("button");
  const revision = ++state.warehouseRevision;
  button.disabled = true;
  elements.warehouseResult.classList.add("hidden");
  try {
    const office = document.querySelector("#officeTransferFile").files[0];
    const warehouse = document.querySelector("#warehouseTransferFile").files[0];
    actions.assertDifferentTyumenFiles(office, warehouse);
    const plan = buildTyumenWarehousePlan({
      officeWorkbook: await actions.loadWorkbook(office), warehouseWorkbook: await actions.loadWorkbook(warehouse),
      officeFileName: office.name, warehouseFileName: warehouse.name, brand: elements.warehouseBrand.value,
    });
    if (revision !== state.warehouseRevision) return;
    state.warehousePlan = plan;
    elements.warehouseRows.innerHTML = plan.map((row, index) => `<tr><td>${actions.escapeHtml(row.article)}</td><td>${actions.escapeHtml(row.name)}</td><td>${row.officeStock}</td><td>${row.warehouseStock}</td><td>${row.target}</td><td><input class="warehouse-quantity" aria-label="${actions.escapeHtml(`Переместить: ${row.name}`)}" type="number" min="0" max="${row.warehouseStock}" step="1" value="${row.quantity}" data-index="${index}" /></td><td data-office-after>${row.officeStock + row.quantity}</td><td data-warehouse-after>${row.warehouseStock - row.quantity}</td></tr>`).join("");
    elements.warehouseResult.classList.remove("hidden");
    elements.warehouseDownload.disabled = !plan.length;
  } catch (error) {
    state.warehousePlan = [];
    alert(error.message);
  } finally { button.disabled = false; }
});
}

export function bindWarehouseRowsInput(context) {
  const {elements, state, actions} = context;
elements.warehouseRows.addEventListener("input", (event) => {
  const input = event.target.closest(".warehouse-quantity");
  if (!input) return;
  const row = state.warehousePlan[Number(input.dataset.index)];
  const value = input.value === "" ? 0 : Number(input.value);
  const valid = Number.isInteger(value) && value >= 0 && value <= row.warehouseStock;
  input.setCustomValidity(valid ? "" : "Количество должно быть целым, неотрицательным и не больше остатка склада.");
  input.closest("tr").querySelector("[data-office-after]").textContent = valid ? row.officeStock + value : "—";
  input.closest("tr").querySelector("[data-warehouse-after]").textContent = valid ? row.warehouseStock - value : "—";
  row.quantity = valid ? value : NaN;
});
}

export function bindWarehouseDownloadClick(context) {
  const {elements, state, actions} = context;
elements.warehouseDownload.addEventListener("click", async () => {
  for (const input of elements.warehouseRows.querySelectorAll("input")) if (!input.reportValidity()) return;
  const items = state.warehousePlan.filter((row) => row.quantity > 0);
  if (!items.length) { alert("Нет позиций для перемещения."); return; }
  elements.warehouseDownload.disabled = true;
  try {
    const bytes = await actions.transferWorkbookBytes({ city: { warehouse: "Склад Тюмень" }, items });
    if (state.warehouseDownloadUrl) URL.revokeObjectURL(state.warehouseDownloadUrl);
    state.warehouseDownloadUrl = URL.createObjectURL(new Blob([bytes], { type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" }));
    actions.triggerDownload(state.warehouseDownloadUrl, `${elements.warehouseBrand.selectedOptions[0].textContent} Заказ на перемещение Склад Тюмень - Офис Тюмень ${actions.todayRu()}.xlsx`);
  } catch (error) { alert(error.message); }
  finally { elements.warehouseDownload.disabled = false; }
});
}
