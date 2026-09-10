import assert from "node:assert/strict";
import { recalculateOrderTable } from '../src/workbookProcessor.js';
import { utils, write, read } from "xlsx";
import { mkdir, writeFile } from "node:fs/promises";
import { unzipSync, zipSync, strFromU8, strToU8 } from "fflate";
import { loadXlsx, saveXlsx, mergeTyumenSources, buildTyumenWarehousePlan, warehouseTransferQuantity, northTyumenFreeStock, buildNorthOrderFiles, finalizeNorthOrderFiles, fillWorkbook } from "../src/workbookProcessor.js";

function book(rows, name = "Тюмень") {
  const wb = utils.book_new();
  utils.book_append_sheet(wb, utils.aoa_to_sheet(rows), name);
  return loadXlsx(write(wb, { type: "buffer", bookType: "xlsx" }));
}
const months = ["сентябрь 2025", "октябрь 2025", "ноябрь 2025", "декабрь 2025", "январь 2026", "февраль 2026", "март 2026", "апрель 2026", "май 2026", "июнь 2026", "июль 2026", "август 2026"];
function source(items) {
  return book([
    ["Тюмень", "Период: 01.09.2025 - 31.08.2026", "Прошлый период: 01.09.2025 - 30.11.2025"],
    ["", "", ...months, "Итого"],
    ["Артикул", "Товар", ...months.map(() => "Количество"), "Количество", "Сумма выручки", "% выручки", "Кумулятивный %", "Категория", "Среднее за месяц", "Количество прошлый период", "Целевой запас", "Остаток", "В пути", "Рекомендованный заказ", "Заказано по факту", "Комментарий"],
    ...items.map(({ article, name, sales = 10, revenue = 100, stock = 0, transit = 0 }) => [article, name, ...months.map(() => sales), 999, revenue, 999, 999, "A+", 999, 12, 999, stock, transit, 999, 999, "Старый расчет"]),
  ]);
}
const office = source([
  { article: "P1", name: "Крем 50 мл", stock: 10, sales: 10, revenue: 100 },
  { article: "P2", name: "Гель 100 мл", stock: 35, sales: 0, revenue: 900 },
  { article: "P3", name: "Только офис", stock: 100, sales: 0, revenue: 0 },
]);
const warehouse = source([
  { article: "P1", name: "Крем 50 мл", stock: 90, sales: 20, revenue: 200 },
  { article: "P2", name: "Гель 100 мл", stock: 65, sales: 0, revenue: 0 },
  { article: "P4", name: "Только склад", stock: 20, sales: 0, revenue: 0 },
]);
const options = { officeWorkbook: office, warehouseWorkbook: warehouse, brand: "skin_synergy" };
for(const second of [null,warehouse]) {
  const recalculated=recalculateOrderTable({workbook:office,warehouseWorkbook:second,brand:'skin_synergy',orderMonth:'2026-10',fileName:'Тюмень.xlsx',warehouseFileName:'Тюмень склад.xlsx'});
  const item=recalculated.rows.find(r=>r.article==='P1');
  assert.equal(item.orderedFact,second?1998:999);
  assert.equal(item.sourceComment,'Старый расчет');
  assert.equal(item.stock,second?100:10);
  assert.notEqual(item.recommended,999);
}
const merged = mergeTyumenSources(options);
const decoded = read(saveXlsx(merged), { type: "buffer" });
const rows = utils.sheet_to_json(decoded.Sheets[decoded.SheetNames[0]], { header: 1, defval: "" });
const p1 = rows.find((r) => r[0] === "P1");
assert.equal(p1[2], 30);
assert.equal(p1[14], 360);
assert.equal(p1[15], 300);
assert.equal(p1[18], "C");
assert.equal(p1[21], 37.5);
assert.equal(p1[22], 100);
assert.equal(p1[24], 0);
assert.equal(p1[25], "");
assert.equal(p1[27], 10);
assert.equal(p1[28], 90);
assert.equal(rows.filter((r) => /^P\d$/.test(r[0])).length, 4);
assert.equal(utils.sheet_to_json(read(saveXlsx(office), { type: "buffer" }).Sheets.Тюмень, { header: 1 })[3][24], 999, "Inputs must not be mutated");
const plan = buildTyumenWarehousePlan(options);
assert.equal(plan.find((r) => r.article === "P1").quantity, 15);
assert.equal(plan.find((r) => r.article === "P2").quantity, 0);
assert.equal(plan.find((r) => r.article === "P3").quantity, 0);
assert.equal(plan.find((r) => r.article === "P4").quantity, 5);
assert.equal(warehouseTransferQuantity(0, 2), 1);
assert.equal(warehouseTransferQuantity(0, 1), 0);
assert.equal(northTyumenFreeStock(100, 0, 0, 20, 10, 0), 10, "Office surplus must not go north");
assert.equal(northTyumenFreeStock(100, 0, 0, 110, 90, 0), 0, "Protect Tyumen target");
assert.equal(northTyumenFreeStock(100, 0, 10, 110, 90, 0), 0);
assert.equal(northTyumenFreeStock(300, 0, 0, 100), 200, "Legacy calculation stays supported");
for(const items of [[{article:'P1',name:'Крем 230 мл'}],[{article:'P1',name:'Крем'},{article:'P1',name:'Крем'}]]) {
  const reviewed=mergeTyumenSources({...options,warehouseWorkbook:source(items)});
  const sheet=read(saveXlsx(reviewed),{type:'buffer'}).Sheets.Тюмень;
  const preserved=utils.sheet_to_json(sheet,{header:1}).filter(r=>r[0]==='P1');
  assert.equal(preserved.length,items.length+1);
  assert.ok(preserved.every(r=>String(r[26]).includes('Проверить:')));
  assert.equal(preserved.reduce((s,r)=>s+r[22],0),10);
}
assert.throws(() => mergeTyumenSources({ ...options, warehouseWorkbook: merged }), /уже объединённая/);
const deficit = mergeTyumenSources({ officeWorkbook: source([{ article: "P1", name: "Крем", sales: 10, stock: 0 }]), warehouseWorkbook: source([{ article: "P1", name: "Крем", sales: 10, stock: 0 }]), brand: "novacutan" });
const deficitRow = utils.sheet_to_json(read(saveXlsx(deficit), { type: "buffer" }).Sheets.Тюмень, { header: 1 })[3];
assert.equal(deficitRow[24], 35, "Combined sales must drive a fresh recommendation with brand coefficient");
const northSource = mergeTyumenSources({ officeWorkbook: source([{ article: "P1", name: "Крем 50 мл", sales: 0, stock: 90 }]), warehouseWorkbook: source([{ article: "P1", name: "Крем 50 мл", sales: 0, stock: 10 }]), brand: "skin_synergy" });
const north = buildNorthOrderFiles([{ fileName: "Skin Synergy Сургут.xlsx", workbook: book([["Skin Synergy Сургут"], ["Артикул", "Наименование", "Количество"], ["P1", "Крем 50 мл", 20]], "Бланк") }], { brand: "skin_synergy", tyumenSourceWorkbook: northSource });
assert.equal(north.planRows[0].fromTyumen, 10);
assert.equal(north.planRows[0].supplierNeed, 10);
const finalized = finalizeNorthOrderFiles(north);
assert.equal(finalized.transfers[0].items[0].quantity, 20);
const finalRows = utils.sheet_to_json(read(saveXlsx(finalized.summaryWorkbook), { type: "buffer" }).Sheets.Бланк, { header: 1 });
assert.equal(finalRows[2][2], 10, "Final supplier export must preserve the warehouse-only stock limit");
const blank = book([["Skin Synergy"], ["Артикул", "Наименование", "Количество"], ["P1", "Крем 50 мл", ""], ["P2", "Гель 100 мл", ""], ["P3", "Только офис", ""], ["P4", "Только склад", ""]], "Бланк");
const filled = fillWorkbook({ sourceWorkbook: loadXlsx(saveXlsx(merged)), sourceFileName: "Тюмень общая.xlsx", blankWorkbook: blank, orderMonth: "2026-10", brand: "skin_synergy" });
assert.equal(filled.summary.sourceCity, "Тюмень");
assert.equal(filled.reportRows.find((r) => r.article === "P1" || r.articleRaw === "P1")?.inserted ?? null, null);
const chzMerged = mergeTyumenSources({ officeWorkbook: source([{ article: "P1", name: "ЧЗ Крем 50 мл", stock: 10 }]), warehouseWorkbook: source([{ article: "P1", name: "Крем 50 мл", stock: 90 }]), brand: "skin_synergy" });
assert.equal(utils.sheet_to_json(read(saveXlsx(chzMerged), { type: "buffer" }).Sheets.Тюмень, { header: 1 })[3][22], 100);
assert.equal(utils.sheet_to_json(read(saveXlsx(chzMerged), { type: "buffer" }).Sheets.Тюмень, { header: 1 })[3][1], "ЧЗ + Крем 50 мл");
const chzWithin = mergeTyumenSources({ officeWorkbook: source([{ article: "P1", name: "ЧЗ Крем 50 мл", stock: 10 }, { article: "P1", name: "Крем 50 мл", stock: 5 }]), warehouseWorkbook: source([{ article: "P1", name: "Крем 50 мл", stock: 90 }]), brand: "skin_synergy" });
const chzRow = utils.sheet_to_json(read(saveXlsx(chzWithin), { type: "buffer" }).Sheets.Тюмень, { header: 1 })[3];
assert.equal(chzRow[0], "P1", "CHZ merging must not overwrite an article in the first column");
assert.equal(chzRow[22], 105);
const nameOnlyPlan = buildTyumenWarehousePlan({ officeWorkbook: source([{ article: "", name: "Novacutan SBIO, 2 мл", stock: 10 }]), warehouseWorkbook: source([{ article: "", name: "Novacutan SBIO, 2 мл", stock: 90 }, { article: "", name: "Novacutan YBIO, 2 мл", stock: 20 }]), brand: "novacutan" });
assert.equal(nameOnlyPlan.length, 2);
assert.equal(nameOnlyPlan.find((r) => r.name.includes("YBIO")).quantity, 5);
const missingArticlePlan = buildTyumenWarehousePlan({ officeWorkbook: source([{ article: "P1", name: "Крем 50 мл", stock: 10 }]), warehouseWorkbook: source([{ article: "", name: "Крем 50 мл", stock: 90 }]), brand: "skin_synergy" });
assert.equal(missingArticlePlan.length, 1);
assert.equal(missingArticlePlan[0].article, "P1");
assert.equal(missingArticlePlan[0].quantity, 15);
await mkdir("test-output/tyumen", { recursive: true });
function inventorySource(items) {
  const rows = utils.sheet_to_json(read(saveXlsx(source(items)), { type: "buffer" }).Sheets.Тюмень, { header: 1, defval: "" });
  rows[0] = ["СКЛАД ДОСТАВКА"];
  rows.forEach((row, index) => { if (index >= 3) row[14] = 0; if (index > 0) row.splice(2, 12); });
  return book(rows, "СКЛАД ДОСТАВКА");
}
const inventory = inventorySource([
  { article: "P1", name: "Крем 50 мл", revenue: 0, stock: 40, transit: 3 },
  { article: "P1", name: "ЧЗ Крем 50 мл", revenue: 0, stock: 50, transit: 2 },
  { article: "P5", name: "Новый товар склада", revenue: 0, stock: 20 },
]);
const inventoryOptions = { ...options, warehouseWorkbook: inventory };
const inventoryMerged = mergeTyumenSources(inventoryOptions);
const inventoryRows = utils.sheet_to_json(read(saveXlsx(inventoryMerged), { type: "buffer" }).Sheets.Тюмень, { header: 1 });
const inventoryP1 = inventoryRows.find((row) => row[0] === "P1");
assert.equal(inventoryP1[1], "ЧЗ + Крем 50 мл");
assert.equal(inventoryP1[2], 10);
assert.equal(inventoryP1[14], 120);
assert.equal(inventoryP1[22], 100);
assert.equal(inventoryP1[23], 5);
assert.equal(inventoryP1[18], "C");
assert.equal(inventoryP1[21], 12.5);
assert.equal(inventoryRows.find((row) => row[0] === "P5")[2], 0);
assert.equal(buildTyumenWarehousePlan(inventoryOptions).find((row) => row.article === "P1").quantity, 15);
assert.throws(() => mergeTyumenSources({ ...options, warehouseWorkbook: inventorySource([{ article: "P1", name: "Крем", revenue: 100 }]) }), /указаны продажи/);
assert.throws(() => mergeTyumenSources({ ...options, officeWorkbook: inventory }), /месячные продажи/);
await writeFile("test-output/tyumen/СКЛАД ДОСТАВКА без продаж.xlsx", saveXlsx(inventory));
await writeFile("test-output/tyumen/Офис Тюмень.xlsx", saveXlsx(office));
await writeFile("test-output/tyumen/Склад Тюмень.xlsx", saveXlsx(warehouse));
await writeFile("test-output/tyumen/Skin Synergy бланк.xlsx", saveXlsx(blank));
await writeFile("test-output/tyumen/Офис Тюмень Север.xlsx", saveXlsx(source([{ article: "P1", name: "Крем 50 мл", sales: 0, stock: 90 }])));
await writeFile("test-output/tyumen/Склад Тюмень Север.xlsx", saveXlsx(source([{ article: "P1", name: "Крем 50 мл", sales: 0, stock: 10 }])));
await writeFile("test-output/tyumen/Skin Synergy Сургут.xlsx", saveXlsx(book([["Skin Synergy Сургут"], ["Артикул", "Наименование", "Количество"], ["P1", "Крем 50 мл", 20]], "Бланк")));
console.log("Tyumen merge, fresh categories/recommendations, warehouse transfers, north export: passed");
// Exercise the valid prefixed XML produced by some Excel exporters.
const prefixFixture = book([["Артикул", "Наименование", "Количество", "Сумма"], ["P1", "Крем 50 мл", 7, 70]], "Бланк");
const packageFiles = unzipSync(saveXlsx(prefixFixture));
for (const path of Object.keys(packageFiles).filter((path) => path === "xl/workbook.xml" || path.startsWith("xl/worksheets/") && path.endsWith(".xml"))) {
  const xml = strFromU8(packageFiles[path]).replace(/<(\/?)([A-Za-z][\w.-]*)(?=[\s/>])/g, "<$1s:$2").replace('xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"', 'xmlns:s="http://schemas.openxmlformats.org/spreadsheetml/2006/main"');
  packageFiles[path] = strToU8(xml);
}
const prefixed = loadXlsx(zipSync(packageFiles));
assert.equal(prefixed.sheets.length, 1);
const prefixFilled = fillWorkbook({ sourceWorkbook: loadXlsx(saveXlsx(merged)), blankWorkbook: prefixed, brand: "skin_synergy", orderMonth: "2026-10" });
const prefixRoundtrip = read(saveXlsx(prefixFilled.blankWorkbook), { type: "buffer" });
assert.equal(prefixRoundtrip.Sheets.Бланк.C2?.v ?? null, null);
assert.equal(prefixRoundtrip.Sheets.Бланк.D2.v, 70);
assert.equal(loadXlsx(saveXlsx(prefixFilled.blankWorkbook)).sheets[0].cells.size, 8);
console.log("Prefixed workbook XML read/write: passed");
