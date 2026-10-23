import { buildWarehouseBalanceResult } from './warehouseBalancePlan.js';
import { utils, write } from 'xlsx';

export function installWarehouseBalance({ loadWorkbook, activate }) {
  const section = document.createElement('section');
  section.id = 'balanceSection'; section.className = 'hidden';
  section.innerHTML = `<h2>Между складами</h2><form class="upload-grid">
    <label class="dropzone"><span class="label">Склад Тюмень</span><input data-tyumen type="file" accept=".xlsx,.xlsm,.xls" required><span data-tyumen-name></span></label>
    <label class="dropzone"><span class="label">Склад Доставка</span><input data-delivery type="file" accept=".xlsx,.xlsm,.xls" required><span data-delivery-name></span></label>
    <button class="download" type="submit">Рассчитать перемещение</button></form>
    <p role="status" data-status></p><div data-result hidden><button type="button" class="download" data-download>Скачать Excel</button>
    <label class="search-field"><span>Поиск</span><input type="search" data-search placeholder="Название или артикул" aria-label="Поиск по названию или артикулу"></label>
    <p data-empty hidden role="status">Ничего не найдено</p>
    <div class="table-wrap"><table><thead><tr><th>Артикул</th><th>Название</th><th>Остаток Доставки</th><th>Остаток Тюмени</th><th>К перемещению</th></tr></thead><tbody></tbody></table></div></div>`;
  document.querySelector('.workspace').append(section);
  const button = document.createElement('button'); button.type = 'button';
  button.id = 'balanceModeButton'; button.className = 'mode-button'; button.textContent = 'Между складами';
  button.onclick = () => activate('balance'); document.querySelector('.mode-switch').append(button);
  const $ = selector => section.querySelector(selector);
  let rows = [];
  let brandLabel = '';
  const searchable = value => String(value ?? '').toLocaleLowerCase('ru').replace(/ё/g, 'е').replace(/\s+/g, ' ').trim();
  $('[data-search]').oninput = () => {
    const query = searchable($('[data-search]').value);
    let visible = 0;
    for (const tr of $('tbody').rows) {
      tr.hidden = !tr.dataset.searchText.includes(query);
      if (!tr.hidden) visible++;
    }
    $('[data-empty]').hidden = visible !== 0;
  };
  $('form').onchange = () => {
    rows = []; $('[data-result]').hidden = true; $('[data-status]').textContent = '';
    for (const kind of ['delivery', 'tyumen']) $(`[data-${kind}-name]`).textContent = $(`[data-${kind}]`).files[0]?.name || '';
  };
  $('form').onsubmit = async event => {
    event.preventDefault(); $('[data-result]').hidden = true;
    const controls = [...$('form').querySelectorAll('input,select,button')];
    controls.forEach(control => control.disabled = true);
    try {
      $('[data-status]').textContent = 'Рассчитываю…';
      const delivery = $('[data-delivery]').files[0], tyumen = $('[data-tyumen]').files[0];
      if (delivery.name === tyumen.name && delivery.size === tyumen.size && delivery.lastModified === tyumen.lastModified) throw Error('Выбрана одна и та же таблица для двух складов.');
      const result = buildWarehouseBalanceResult({ deliveryWorkbook: await loadWorkbook(delivery, { allowLegacyXls: true }), tyumenWorkbook: await loadWorkbook(tyumen, { allowLegacyXls: true }), deliveryFileName: delivery.name, tyumenFileName: tyumen.name });
      rows = result.rows; brandLabel = result.brandLabel;
      $('tbody').replaceChildren();
      $('[data-search]').value = ''; $('[data-empty]').hidden = true;
      for (const row of rows) {
        const tr = document.createElement('tr');
        tr.dataset.searchText = searchable(`${row.article} ${row.name}`);
        for (const value of [row.article, row.name, row.deliveryStock, row.tyumenStock]) {
          const td = document.createElement('td'); td.textContent = value ?? ''; tr.append(td);
        }
        const td = document.createElement('td'), input = document.createElement('input');
        input.type = 'number'; input.className = 'warehouse-quantity'; input.min = '0'; input.step = '1'; input.value = row.quantity ?? '';
        input.setAttribute('aria-label', `К перемещению: ${row.name}`);
        if (row.deliveryStock !== null && row.deliveryStock >= 0) input.max = Math.floor(row.deliveryStock);
        input.oninput = () => {
          const quantity = input.value === '' ? null : Number(input.value);
          const valid = quantity === null || (Number.isInteger(quantity) && quantity >= 0 && (row.deliveryStock === null || quantity <= row.deliveryStock));
          input.setCustomValidity(valid ? '' : 'Укажите целое неотрицательное количество не больше остатка Доставки.');
          if (valid) row.quantity = quantity;
          $('[data-download]').disabled = [...$('tbody').querySelectorAll('input')].some(el => !el.validity.valid);
        };
        td.append(input);
        if (row.warning) { const warning = document.createElement('p'); warning.className = 'warn'; warning.textContent = row.warning; td.append(warning); }
        tr.append(td); $('tbody').append(tr);
      }
      $('[data-download]').disabled = false; $('[data-result]').hidden = false;
      $('[data-status]').textContent = `Позиций: ${rows.length}. Для проверки: ${rows.filter(row => row.warning).length}.`;
    } catch (error) { $('[data-status]').textContent = error.message; }
    finally { controls.forEach(control => control.disabled = false); }
  };
  $('[data-download]').onclick = () => {
    const workbook = utils.book_new();
    const exportRows = [
      ...rows.filter(row => !row.warning && row.quantity !== 0),
      ...rows.filter(row => row.warning),
    ];
    const sheet = utils.aoa_to_sheet([['Артикул', 'Название', 'К перемещению', 'Комментарий'], ...exportRows.map(row => [row.article, row.name, row.quantity, row.warning || ''])]);
    sheet['!cols'] = [{ wch: 20 }, { wch: 90 }, { wch: 20 }, { wch: 72 }];
    exportRows.forEach((row, index) => {
      if (!row.warning) return;
      const address = `B${index + 2}`;
      sheet[address].c = [{ a: 'Проверка', t: row.warning }];
    });
    utils.book_append_sheet(workbook, sheet, 'Между складами');
    const url = URL.createObjectURL(new Blob([write(workbook, { type: 'array', bookType: 'xlsx' })], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }));
    const link = document.createElement('a'); link.href = url; link.download = `${brandLabel} Доставка - Тюмень перемещение.xlsx`; link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
}
