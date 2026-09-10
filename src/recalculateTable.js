import {recalculateOrderTable,saveXlsx} from './workbookProcessor.js';

export function installTableRecalculation({loadWorkbook,activate,brands,month}) {
  const section=document.createElement('section');section.id='recalculateSection';section.className='hidden';
  section.innerHTML=`<h2>Пересчёт таблицы</h2><form class="upload-grid">
  <label class="brand-field"><span class="label">Бренд</span><select data-brand>${brands}</select></label>
  <label class="brand-field"><span class="label">Месяц заказа</span><input type="month" data-month required></label>
  <label class="dropzone"><span class="label">Таблица / офис Тюмени</span><input type="file" data-office accept=".xlsx,.xlsm,.xls" required><span data-office-name></span></label>
  <label class="brand-field"><span class="label">Тюмень: 2 склада</span><input type="checkbox" data-two></label>
  <label class="dropzone hidden" data-warehouse-field><span class="label">СКЛАД ДОСТАВКА</span><input type="file" data-warehouse accept=".xlsx,.xlsm,.xls"><span data-warehouse-name></span></label>
  <button type="submit" class="download">Пересчитать таблицу</button></form>
  <p data-status role="status"></p><div data-result hidden><a class="download" data-download>Скачать таблицу</a><h3>Позиции для проверки</h3><div data-review></div></div>`;
  document.querySelector('.workspace').append(section);
  const button=document.createElement('button');button.type='button';button.className='mode-button';button.id='recalculateModeButton';button.textContent='Пересчёт таблицы';
  document.querySelector('.mode-switch').append(button);button.onclick=()=>activate('recalculate');
  const $=s=>section.querySelector(s);$('[data-month]').value=month;
  let url=null;
  function reset(){if(url)URL.revokeObjectURL(url);url=null;$('[data-result]').hidden=true;$('[data-status]').textContent='';}
  $('form').addEventListener('change',()=>{
    reset();$('[data-warehouse-field]').classList.toggle('hidden',!$('[data-two]').checked);$('[data-warehouse]').required=$('[data-two]').checked;
    for(const kind of ['office','warehouse'])$(`[data-${kind}-name]`).textContent=$(`[data-${kind}]`).files[0]?.name||'';
  });
  $('form').onsubmit=async event=>{
    event.preventDefault();reset();const submit=$('button[type=submit]');submit.disabled=true;
    // Freeze input controls during the async read so files and options cannot diverge.
    const inputs=[...section.querySelectorAll('input,select')];inputs.forEach(e=>e.disabled=true);
    try {
      $('[data-status]').textContent='Пересчитываю…';
      const file=$('[data-office]').files[0],second=$('[data-two]').checked?$('[data-warehouse]').files[0]:null;
      if(second&&file.name===second.name&&file.size===second.size&&file.lastModified===second.lastModified)throw Error('Выбрана одна и та же таблица для двух складов.');
      const result=recalculateOrderTable({workbook:await loadWorkbook(file,{allowLegacyXls:true}),warehouseWorkbook:second?await loadWorkbook(second,{allowLegacyXls:true}):null,brand:$('[data-brand]').value,orderMonth:$('[data-month]').value,fileName:file.name,warehouseFileName:second?.name});
      url=URL.createObjectURL(new Blob([saveXlsx(result.workbook)],{type:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'}));
      $('[data-download]').href=url;$('[data-download]').download=file.name.replace(/\.(xlsx|xlsm|xls)$/i,'')+(second?' общая':'')+' пересчитанная.'+(/\.xlsm$/i.test(file.name)?'xlsm':'xlsx');
      const review=result.rows.filter(r=>r.review);$('[data-review]').replaceChildren();
      for(const row of review){const p=document.createElement('p');p.className='warn';p.textContent=`Строка ${row.rowIndex}: ${row.articleRaw} ${row.name}. ${row.sourceComment || 'Неоднозначная позиция, данные сохранены отдельно.'}`;$('[data-review]').append(p);}
      if(!review.length)$('[data-review]').textContent='Неоднозначных позиций нет.';
      $('[data-result]').hidden=false;$('[data-status]').textContent=`Пересчитано позиций: ${result.rows.length}. Для проверки: ${review.length}. Фактический заказ и комментарии сохранены.`;
    }catch(e){$('[data-status]').textContent=e.message;}finally{submit.disabled=false;inputs.forEach(e=>e.disabled=false);}
  };
}
