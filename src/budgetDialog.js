import { planBudget, coverage, discountValue } from './budgetPlanner.js';

const escape = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const money = value => Number(value).toLocaleString('ru-RU', { maximumFractionDigits: 2 });
const groupLabel = value => value === 'main' ? 'Бланк' : value.toUpperCase();

export function openBudgetDialog({ rows, christina, apply, fixedPricing=false, initialTargets={} }) {
  // Work from a snapshot. Only the explicit Apply action updates the live order.
  if (!rows.length) { alert('Нет позиций для перерасчета.'); return; }
  const groups = [...new Set(rows.map(r => r.group))];
  const dialog = document.createElement('dialog');
  dialog.className = 'budget-dialog';
  dialog.innerHTML = `<form method="dialog"><header><h2>Заказ до суммы</h2><button aria-label="Закрыть">×</button></header></form>
    <section class="budget-settings">
      <label><input type="radio" name="price-mode" value="net" checked> Цена уже с нашей скидкой</label>
      <label><input type="radio" name="price-mode" value="gross"> Применить нашу скидку</label>
      <label data-discount hidden>Наша скидка, % <input data-discount-value value="30" inputmode="decimal"></label>
      <div data-prices></div>
      ${christina ? '<label><input type="checkbox" data-separate> Отдельные суммы HOME и PROFF</label>' : ''}
      <div data-targets></div>
    </section>
    <p data-summary aria-live="polite"></p>
    <p data-warning role="alert"></p>
    <div class="budget-scroll"><table><thead><tr><th>Закрепить</th><th>Позиция</th><th>Категория</th><th>Цена закупки</th><th>Сейчас</th><th>После</th><th>Запас, мес.</th><th>Комментарий</th></tr></thead><tbody></tbody></table></div>
    <footer><button type="button" data-run>Рассчитать</button><button type="button" data-permission hidden>Все равно продолжить</button><button type="button" data-apply disabled>Применить</button><button type="button" data-back>Назад</button></footer>`;
  document.body.append(dialog);
  const $ = s => dialog.querySelector(s);
  let preview = null, permissions = {}, pending = '';
  const locks = new Set(rows.filter(r => r.locked || r.excluded).map(r => r.key));
  $('[data-prices]').innerHTML = groups.map((g,index) => {
    const prices = new Map(rows.filter(r => r.group === g).flatMap(r => r.prices || []).map(p => [p.id,p]));
    return `<label>${escape(groupLabel(g))}: колонка цены <select data-price-group="${index}"><option value="">Выберите колонку</option>${[...prices.values()].map(p => `<option value="${escape(p.id)}">${escape(p.label)}</option>`).join('')}</select></label>`;
  }).join('');
  function targets() {
    $('[data-targets]').innerHTML = ($('[data-separate]')?.checked ? groups : ['Общая сумма']).map((g,i) => `<label>${escape(g.toUpperCase())}, ₽ <input type="number" min="0" step="0.01" data-target="${i}" value="${escape(initialTargets[g] ?? initialTargets[groups[i]] ?? '')}" inputmode="decimal"></label>`).join('');
  }
  function pricedRows() {
    if(fixedPricing)return rows.map(r=>({...r,locked:locks.has(r.key),excluded:locks.has(r.key)}));
    const discount = $('input[name="price-mode"]:checked').value === 'gross' ? discountValue($('[data-discount-value]').value) : 0;
    return rows.map(r => {
      const col = $(`[data-price-group="${groups.indexOf(r.group)}"]`).value;
      const selected = r.prices?.find(p => p.id === col);
      const price = selected?.price;
      return { ...r, price: price > 0 ? Math.round(price * (1-discount/100)*100)/100 : 0,
        pricing: { column: selected?.column, label: selected?.label, basePrice: price, discount },
        locked: locks.has(r.key), excluded: locks.has(r.key) };
    });
  }
  function draw(data) {
    $('tbody').innerHTML = data.map(r => `<tr>
      <td><input type="checkbox" data-lock="${escape(r.key)}" aria-label="Закрепить ${escape(r.name)}" ${locks.has(r.key) ? 'checked' : ''}></td>
      <td>${escape(r.name)}</td><td>${escape(r.category || 'Нет данных')}</td>
      <td>${r.price ? money(r.price) : 'Нет цены'}</td>
      <td>${money(r.before ?? r.quantity)}</td><td>${money(r.quantity)}</td>
      <td>${coverage(r,r.quantity) == null ? 'Нет продаж' : money(coverage(r,r.quantity))}</td>
      <td>${escape(r.quantity > r.before ? additionComment(r.quantity-r.before,r.unit) : r.unsafe ? 'Повтор или неоднозначное соответствие: только вручную' : r.demand > 0 ? '' : 'Только ручное изменение')}</td>
    </tr>`).join('');
  }
  function reset() {
    // A changed price, target or lock invalidates both preview and exceptions.
    preview=null; pending=''; permissions={}; $('[data-apply]').disabled=true; $('[data-permission]').hidden=true; $('[data-warning]').textContent='';
    try { const data=pricedRows(); draw(data); $('[data-summary]').textContent=`Текущий заказ: ${money(data.reduce((s,r)=>s+r.quantity*r.price,0))} ₽`; } catch(e) { $('[data-warning]').textContent=e.message; }
  }
  function run() {
    try {
      const data=pricedRows();
      if(!fixedPricing)for(const select of dialog.querySelectorAll('[data-price-group]')) if(!select.value) throw Error('Выберите колонку цены для каждого бланка.');
      const split=$('[data-separate]')?.checked;
      const plans=(split ? groups : ['all']).map((g,i)=>{
        const field=$(`[data-target="${i}"]`);
        if(field.value.trim()==='') throw Error('Введите целевую сумму в рублях.');
        return planBudget(split ? data.filter(r=>r.group===g) : data, Number(field.value), permissions);
      });
      preview=plans.flatMap(p=>p.rows); draw(preview);
      $('[data-summary]').textContent=plans.map((p,i)=>`${split ? groups[i].toUpperCase()+': ' : ''}${money(p.before)} → ${money(p.total)} ₽ (цель ${money(p.target)} ₽)`).join('; ');
      const blocked=plans.find(p=>!p.complete);
      pending=blocked?.reason || '';
      $('[data-warning]').textContent=pending==='overSix' ? 'Все доступные шаги закупки превышают запас на 6 месяцев, включая доставку. Разрешить превышение?' : pending==='belowOne' ? 'Дальнейшее сокращение уменьшит запас ниже 1 месяца + доставка. Разрешить?' : pending;
      $('[data-permission]').hidden=!['overSix','belowOne'].includes(pending);
      $('[data-apply]').disabled=Boolean(blocked);
    } catch(e) { preview=null; $('[data-apply]').disabled=true; $('[data-warning]').textContent=e.message; }
  }
  dialog.addEventListener('change', event=>{
    if(event.target.matches('[data-lock]')) {
      event.target.checked ? locks.add(event.target.dataset.lock) : locks.delete(event.target.dataset.lock);
    }
    if(event.target.matches('[data-separate]')) targets();
    $('[data-discount]').hidden=$('input[name="price-mode"]:checked').value!=='gross';
    reset();
  });
  dialog.addEventListener('input', event=>{ if(event.target.matches('[data-target], [data-discount-value]')) { preview=null; permissions={}; $('[data-apply]').disabled=true; $('[data-permission]').hidden=true; } });
  $('[data-run]').onclick=run;
  $('[data-permission]').onclick=()=>{ if(pending==='overSix') permissions.allowOverSix=true; if(pending==='belowOne') permissions.allowBelowOne=true; run(); };
  $('[data-back]').onclick=()=>dialog.close();
  $('[data-apply]').onclick=()=>{
    if (!preview) return;
    try { apply(preview,locks); dialog.close(); }
    catch (error) { $('[data-warning]').textContent=error.message; }
  };
  dialog.addEventListener('close',()=>dialog.remove(),{once:true});
  if(christina){$('[data-separate]').checked=true;if(fixedPricing)$('[data-separate]').parentElement.hidden=true;}
  if(fixedPricing){
    $('[data-prices]').hidden=true;
    for(const el of dialog.querySelectorAll('input[name="price-mode"]'))el.parentElement.hidden=true;
  }
  targets(); reset(); dialog.showModal();
}

export function additionComment(quantity, unit) {
  return `Добавилось ${money(quantity)} ${unit > 1 ? 'уп.' : 'шт.'} Для закупа до суммы.`;
}
