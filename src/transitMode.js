import {parseInvoiceExcel,parseInvoicePdf,buildTransitPlan,applyTransitPlan} from './transitInvoices.js';
import {saveXlsx,sheetBounds,sheetCellValue} from './workbookProcessor.js';
import {utils,write} from 'xlsx';

export function installTransitMode({loadWorkbook,activate}){
  const section=document.createElement('section');section.id='transitSection';section.className='hidden';
  section.innerHTML=`<h2>Товар в пути · ANGIOPHARM</h2><form class="upload-grid">
    <label class="dropzone"><span class="icon">↑</span><span class="label">Актуальная таблица</span><span class="filename" data-source-name>.xlsx, .xlsm или .xls</span><input data-source type="file" accept=".xlsx,.xlsm,.xls" required></label>
    <label class="dropzone"><span class="icon">↑</span><span class="label">Оплаченные счета</span><span class="filename" data-invoice-name>PDF или Excel</span><input data-invoices type="file" accept=".pdf,.xlsx,.xlsm,.xls" multiple required></label>
    <button type="submit" class="primary">Заполнить «В пути»</button></form>
    <p data-status role="status"></p><div data-result hidden><button type="button" class="download" data-download>Скачать таблицу</button> <button type="button" class="download" data-missing>Скачать несопоставленные</button>
    <div class="table-wrap"><table><thead><tr><th>Артикул</th><th>Товар из счёта</th><th>Количество в пути</th><th>Строка таблицы</th><th>Проверка</th></tr></thead><tbody></tbody></table></div></div>`;
  document.querySelector('.workspace').append(section);
  const button=document.createElement('button');button.type='button';button.className='mode-button';button.id='transitModeButton';button.textContent='Товар в пути';button.onclick=()=>activate('transit');document.querySelector('.mode-switch').append(button);
  const $=s=>section.querySelector(s);let workbook,plan,sourceName;
  $('form').onchange=()=>{$('[data-result]').hidden=true;plan=null;$('[data-status]').textContent='';$('[data-source-name]').textContent=$('[data-source]').files[0]?.name||'.xlsx, .xlsm или .xls';$('[data-invoice-name]').textContent=[...$('[data-invoices]').files].map(f=>f.name).join(', ')||'PDF или Excel';};
  const download=(bytes,name)=>{const url=URL.createObjectURL(new Blob([bytes],{type:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'}));const a=document.createElement('a');a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);};
  const status=()=>{$('[data-status]').textContent=`Позиций счетов: ${plan.rows.length}. Для ручной проверки: ${plan.rows.filter(r=>r.targetRow===null).length}.`;};
  $('form').onsubmit=async e=>{
    e.preventDefault();const controls=[...$('form').querySelectorAll('input,button')];controls.forEach(x=>x.disabled=true);$('[data-result]').hidden=true;
    try{
      $('[data-status]').textContent='Читаю таблицу и счета…';const source=$('[data-source]').files[0];sourceName=source.name;
      workbook=await loadWorkbook(source,{allowLegacyXls:true});
      let metadata=sourceName;for(const sheet of workbook.sheets)for(let r=1;r<=Math.min(14,sheetBounds(sheet).maxRow);r++)for(let c=1;c<=sheetBounds(sheet).maxColumn;c++)metadata+=' '+sheetCellValue(sheet,r,c);
      if(!/ангиофарм|angiopharm/iu.test(metadata))throw Error('В таблице не подтверждён бренд ANGIOPHARM.');
      const invoices=[],hashes=new Set(),identities=new Set();
      for(const file of $('[data-invoices]').files){
        const bytes=await file.arrayBuffer();const digest=[...new Uint8Array(await crypto.subtle.digest('SHA-256',bytes))].map(x=>x.toString(16).padStart(2,'0')).join('');
        if(hashes.has(digest))throw Error(`Счёт «${file.name}» загружен повторно.`);hashes.add(digest);
        const rows=/\.pdf$/i.test(file.name)?await parseInvoicePdf(bytes,file.name):parseInvoiceExcel(await loadWorkbook(file,{allowLegacyXls:true}),file.name);
        if(rows.invoiceId&&identities.has(rows.invoiceId))throw Error(`Счёт «${file.name}» с тем же номером и датой загружен повторно.`);
        if(rows.invoiceId)identities.add(rows.invoiceId);
        invoices.push({name:file.name,rows});
      }
      plan=buildTransitPlan(workbook,invoices);$('tbody').replaceChildren();
      for(const row of plan.rows){
        const tr=document.createElement('tr');for(const value of [row.article,row.name]){const td=document.createElement('td');td.textContent=value;tr.append(td);}
        const quantityCell=document.createElement('td'),quantity=document.createElement('input');quantity.type='number';quantity.min='0';quantity.step='1';quantity.className='warehouse-quantity';quantity.value=row.quantity;quantity.setAttribute('aria-label',`В пути: ${row.article}`);quantity.oninput=()=>{row.quantity=quantity.value===''?null:Number(quantity.value);};quantityCell.append(quantity);tr.append(quantityCell);
        const targetCell=document.createElement('td'),select=document.createElement('select');select.setAttribute('aria-label',`Соответствие: ${row.article}`);select.add(new Option('Не сопоставлено',''));
        for(const product of plan.products)select.add(new Option(`${product.article} · ${product.name} (строка ${product.row})`,String(product.row)));
        select.value=row.targetRow===null?'':String(row.targetRow);const warning=document.createElement('td');warning.textContent=row.warning;warning.className=row.warning?'warn':'';
        select.onchange=()=>{row.targetRow=select.value?Number(select.value):null;row.targetName=plan.products.find(p=>p.row===row.targetRow)?.name||'';row.warning=row.targetRow===null?'Не сопоставлено. Требуется ручная проверка.':'';warning.textContent=row.warning;status();};targetCell.append(select);tr.append(targetCell,warning);$('tbody').append(tr);
      }
      $('[data-result]').hidden=false;status();
    }catch(error){$('[data-status]').textContent=error.message;plan=null;}finally{controls.forEach(x=>x.disabled=false);}
  };
  $('[data-download]').onclick=()=>{try{if(!plan)return;const count=plan.rows.filter(r=>r.targetRow===null).length;if(count&&!window.confirm(`Не сопоставлено позиций: ${count}. Они не будут внесены в таблицу. Скачайте их отдельным списком для ручной проверки. Продолжить выгрузку таблицы?`))return;const output=applyTransitPlan(workbook,plan);download(saveXlsx(output),sourceName.replace(/\.(xlsx|xlsm|xls)$/i,'')+' с товарами в пути.'+(/\.xlsm$/i.test(sourceName)?'xlsm':'xlsx'));}catch(error){$('[data-status]').textContent=error.message;}};
  $('[data-missing]').onclick=()=>{if(!plan)return;const book=utils.book_new();const sheet=utils.aoa_to_sheet([['Артикул','Название','Количество','Счёт','Комментарий'],...plan.rows.filter(r=>r.targetRow===null).map(r=>[r.article,r.name,r.quantity,[...new Set(r.files)].join('; '),r.warning])]);sheet['!cols']=[{wch:16},{wch:80},{wch:14},{wch:50},{wch:65}];utils.book_append_sheet(book,sheet,'Для проверки');download(write(book,{type:'array',bookType:'xlsx'}),'ANGIOPHARM несопоставленные товары в пути.xlsx');};
}
