import { discountValue } from './budgetPlanner.js';
const escape = v => String(v ?? '').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export const money = v => Number(v).toLocaleString('ru-RU',{minimumFractionDigits:2,maximumFractionDigits:2});

/** File selection owns price settings; results keep a snapshot until recalculated. */
export function createOrderPricing(form, files, load, options, changed) {
  const panel=document.createElement('section');
  panel.className='order-pricing'; panel.hidden=true; form.append(panel);
  let generation=0, groups=[];
  async function refresh() {
    const version=++generation;
    const entries=files();
    groups=[...new Set(entries.map(e=>e.group))];
    panel.hidden=!entries.length; panel.textContent=entries.length?'Читаю колонки цен…':'';
    panel.dataset.ready='false';
    try {
      const sets=[];
      for(const g of groups) {
        const e=entries.find(e=>e.group===g);
        sets.push(await options(await load(e.file),e.file.name));
      }
      if(version!==generation)return;
      panel.innerHTML='<h3>Цена закупки</h3>'+groups.map((g,i)=>`<div data-group="${i}"><strong>${g==='main'?'Бланк':escape(g.toUpperCase())}</strong><label>Колонка цены<select data-column><option value="">Выберите колонку</option>${sets[i].map(p=>`<option value="${escape(p.id)}">${escape(p.label)}${p.price>0?' · '+money(p.price)+' ₽':''}</option>`).join('')}</select></label><label>Скидка<select data-mode><option value="net">Уже учтена в цене</option><option value="gross">Применить нашу скидку</option></select></label><label data-discount-label hidden>Наша скидка, %<input data-discount inputmode="decimal" placeholder="30 или 30%"></label></div>`).join('');
      panel.dataset.ready='true';
    } catch(e) {if(version===generation)panel.textContent=e.message;}
  }
  panel.addEventListener('change',e=>{
    const group=e.target.closest('[data-group]');
    if(group)group.querySelector('[data-discount-label]').hidden=group.querySelector('[data-mode]').value==='net';
    changed();
  });
  panel.addEventListener('input',changed);
  return {refresh, read(){
    if(panel.dataset.ready!=='true')throw Error('Дождитесь чтения колонок цен.');
    return Object.fromEntries(groups.map((g,i)=>{
      const el=panel.querySelector(`[data-group="${i}"]`),column=el.querySelector('[data-column]').value;
      if(!column)throw Error(`Выберите колонку цены: ${g==='main'?'бланк':g.toUpperCase()}.`);
      return [g,{column,discount:el.querySelector('[data-mode]').value==='gross'?discountValue(el.querySelector('[data-discount]').value):0}];
    }));
  }};
}

export function priceOrderRows(rows,settings) {
  return rows.map(r=>{
    const config=settings[r.group],selected=r.prices?.find(p=>p.id===config?.column);
    const price=selected?.price>0?Math.round(selected.price*(1-config.discount/100)*100)/100:0;
    return {...r,price,pricing:{column:selected?.column,label:selected?.label,basePrice:selected?.price,discount:config?.discount || 0}};
  });
}
