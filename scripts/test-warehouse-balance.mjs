import assert from 'node:assert/strict';
import { utils, write } from 'xlsx';
import { loadXlsx } from '../src/workbookProcessor.js';
import { balanceQuantity, buildWarehouseBalance } from '../src/warehouseBalancePlan.js';
function book(rows, warehouse, brand = 'ANGIOPHARM') {
  const workbook = utils.book_new();
  utils.book_append_sheet(workbook, utils.aoa_to_sheet([[brand], [warehouse], ['Артикул', 'Название', 'Остаток'], ...rows]), 'Остатки');
  return loadXlsx(write(workbook, { type: 'array', bookType: 'xlsx' }));
}
const options = (delivery, tyumen) => ({ deliveryWorkbook: book(delivery, 'СКЛАД ДОСТАВКА'), tyumenWorkbook: book(tyumen, 'Склад Тюмень'), deliveryFileName: 'доставка.xlsx', tyumenFileName: 'тюмень.xlsx', brand: 'angiopharm' });
assert.equal(balanceQuantity(200, 25), 20);
assert.equal(balanceQuantity(200, 250), 0);
assert.equal(balanceQuantity(13, 0), 4);
assert.equal(balanceQuantity(200, 25, 70), 133);
assert.equal(balanceQuantity(200, 25, 0), 0);
assert.equal(balanceQuantity(200, 25, 100), 200);
assert.equal(balanceQuantity(200, 25, 30), 43);
assert.throws(() => balanceQuantity(200, 25, 101), /0 до 100/);
assert.equal(balanceQuantity(5, 0), 2);
assert.equal(balanceQuantity(1, 0), 0);
assert.equal(balanceQuantity(6, 0), 4);
for (const percent of [0, 20, 70, 100]) {
  for (const [stock, expected] of [[0, 0], [3, 0], [4, 2], [5, 2], [6, 4], [20, 4]]) {
    for (const tyumen of [0, 1]) {
      const proportional = Math.max(0, Math.ceil((stock + tyumen) * percent / 100 - tyumen - 1e-10));
      const quantity = Math.min(stock, Math.max(expected, proportional));
      assert.equal(balanceQuantity(stock, tyumen, percent), quantity === 1 ? 0 : quantity);
    }
  }
}
assert.equal(balanceQuantity(21, 0), 5);
assert.equal(balanceQuantity(5, 1), 2);
assert.equal(balanceQuantity(13, 1), 4);
assert.equal(balanceQuantity(20, 1), 4);
assert.equal(balanceQuantity(13, 1, 70), 9);
assert.equal(balanceQuantity(13, 0, 100), 13);
assert.equal(balanceQuantity(5, 1, 100), 5);
assert.equal(balanceQuantity(3, 0, 100), 3);
assert.equal(balanceQuantity(5, 2), 0);
assert.equal(balanceQuantity(13, 2), 0);
assert.equal(balanceQuantity(13, 2, 70), 9);
assert.equal(balanceQuantity(21, 1), 4);
assert.equal(balanceQuantity(24, 5), 0);
assert.equal(balanceQuantity(0, 0), 0);
assert.equal(balanceQuantity(null, 0), null);
assert.equal(balanceQuantity(-1, 0), null);
let rows = buildWarehouseBalance(options([['1', 'Крем 50 мл', 150], ['1', 'ЧЗ Крем 50 мл', 50]], [['1', 'Крем 50 мл', 25]]));
assert.equal(rows.length, 1); assert.equal(rows[0].deliveryStock, 200); assert.equal(rows[0].quantity, 20);
rows = buildWarehouseBalance(options([['1', 'Крем 50 мл', 0]], [['1', 'Крем 50 мл', 0]]));
assert.equal(rows[0].quantity, 0);
rows = buildWarehouseBalance(options([['1', 'Крем 50 мл', 10]], [['2', 'Крем 50 мл', 0]]));
assert.equal(rows.length, 2); assert.ok(rows.every(row => row.quantity === null && row.warning));
rows = buildWarehouseBalance(options([['1', 'Крем 50 мл', null]], [['1', 'Крем 50 мл', 0]]));
assert.equal(rows[0].quantity, 0);
const mismatch = options([['1', 'Крем', 20]], [['1', 'Крем', 0]]);
mismatch.tyumenWorkbook = book([['1', 'Крем', 0]], 'Тюмень', 'CHRISTINA');
assert.throws(() => buildWarehouseBalance(mismatch), /бренд/);
const wrongWarehouse = options([['1', 'Крем', 20]], [['1', 'Крем', 0]]);
wrongWarehouse.tyumenWorkbook = book([['1', 'Крем', 0]], 'СКЛАД ДОСТАВКА');
assert.throws(() => buildWarehouseBalance(wrongWarehouse), /склад/);
console.log('Warehouse balance tests passed');
for (const [brand, label] of Object.entries({ angiopharm: 'ANGIOPHARM', christina: 'CHRISTINA', levissime: 'LeviSsime', sothys: 'SOTHYS', novacutan: 'NOVACUTAN', skin_synergy: 'Skin Synergy', klapp: 'KLAPP' })) {
  const input = options([], []);
  input.brand = brand;
  input.deliveryWorkbook = book([['1', 'Крем 50 мл', 200], ['1', 'ЧЗ Крем 50 мл', 0]], 'СКЛАД ДОСТАВКА', label);
  input.tyumenWorkbook = book([['1', 'Крем 50 мл', 25]], 'Склад Тюмень', label);
  const result = buildWarehouseBalance(input);
  assert.equal(result.length, 1, brand); assert.equal(result[0].quantity, 20, brand);
}
const sizes = buildWarehouseBalance(options([['1', 'Крем 50 мл', 200], ['1', 'ЧЗ Крем 500 мл', 100]], [['1', 'Крем 50 мл', 25]]));
assert.equal(sizes.find(row => row.name === 'Крем 50 мл').quantity, 20);
assert.equal(sizes.find(row => row.name === 'ЧЗ Крем 500 мл').quantity, null);
const invalidStock = options([['1', 'Крем', 'неизвестно']], [['1', 'Крем', 0]]);
assert.equal(buildWarehouseBalance(invalidStock)[0].quantity, null);
console.log('All seven brands and ambiguous sizes passed');
const auto = options([['1', 'Крем', 200]], [['1', 'Крем', 25]]);
delete auto.brand;
assert.equal(buildWarehouseBalance(auto)[0].quantity, 20);
auto.tyumenWorkbook = book([['1', 'Крем', 25]], 'Тюмень', 'Без марки');
assert.throws(() => buildWarehouseBalance(auto), /однозначно определить бренд/);
for (const location of ['Склад Сургут', 'Склад Казань', 'Склад Основной', 'Склад Москва', 'Склад Доставка']) {
  const invalid = options([['1', 'Крем', 200]], [['1', 'Крем', 25]]);
  invalid.tyumenWorkbook = book([['1', 'Крем', 25]], location);
  assert.throws(() => buildWarehouseBalance(invalid), /город|склад/i, location);
}
const hiddenCity = options([['1', 'Крем', 200]], [['1', 'Крем', 25]]);
hiddenCity.deliveryWorkbook = book([['1', 'Крем', 200]], 'Склад Доставка Сургут');
assert.throws(() => buildWarehouseBalance(hiddenCity), /другой город/);
const swapped = options([['1', 'Крем', 200]], [['1', 'Крем', 25]]);
swapped.deliveryWorkbook = book([['1', 'Крем', 200]], 'Склад Тюмень');
assert.throws(() => buildWarehouseBalance(swapped), /одинаковых склада/);
const normal = options([['1', 'Крем', 200]], [['1', 'Крем', 25]]);
assert.deepEqual(buildWarehouseBalance(normal), buildWarehouseBalance({
  deliveryWorkbook: normal.tyumenWorkbook, deliveryFileName: normal.tyumenFileName,
  tyumenWorkbook: normal.deliveryWorkbook, tyumenFileName: normal.deliveryFileName,
}));
auto.tyumenWorkbook = book([['1', 'Крем', 25]], 'Тюмень', 'ANGIOPHARM CHRISTINA');
assert.throws(() => buildWarehouseBalance(auto), /однозначно определить бренд/);
