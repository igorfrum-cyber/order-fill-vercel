import { utils, write } from 'xlsx';
import { unzipSync, zipSync, strToU8 } from 'fflate';

export const months = ['сентябрь 2025','октябрь 2025','ноябрь 2025','декабрь 2025','январь 2026','февраль 2026','март 2026','апрель 2026','май 2026','июнь 2026','июль 2026','август 2026'];
export function book(rows, name = 'Тюмень', options = {}) {
  const workbook = utils.book_new();
  const sheet = utils.aoa_to_sheet(rows);
  sheet['!cols'] = [{wch:18},{wch:55}];
  utils.book_append_sheet(workbook, sheet, name);
  return write(workbook, {type:'buffer',bookType:'xlsx', ...options});
}
export function source(items = [{article:'P1',name:'Крем 50 мл',stock:3,sales:10}], {inventory=false} = {}) {
  const rows = [
    ['Тюмень','Период: 01.09.2025 - 31.08.2026','Прошлый период: 01.09.2025 - 30.11.2025'],
    ['', '', ...months, 'Итого'],
    ['Артикул','Товар',...months.map(()=>'Количество'),'Количество','Сумма выручки','% выручки','Кумулятивный %','Категория','Среднее за месяц','Количество прошлый период','Целевой запас','Остаток','В пути','Рекомендованный заказ','Заказано по факту','Комментарий'],
    ...items.map(({article,name,sales=10,stock=0,transit=0,fact='',comment='',revenue=100})=>[article,name,...months.map(()=>sales),sales*12,revenue,0,0,'A',sales,30,20,stock,transit,17,fact,comment]),
  ];
  if (inventory) {
    rows[0] = ['СКЛАД ДОСТАВКА'];
    rows.forEach((row,i)=>{ if(i>=3){row[14]=0;row[15]=0;} if(i>0)row.splice(2,12);});
  }
  return book(rows, inventory ? 'СКЛАД ДОСТАВКА' : 'Тюмень');
}
export function blank(brand='skin_synergy', city='Тюмень', quantity='', article='P1', name='Крем 50 мл') {
  const header = ['Артикул','Наименование','Объем','Цена','Цена со скидкой',['levissime','klapp'].includes(brand)?'Заказ':'Количество',brand==='levissime'?'Кол-во в уп.':'Шт. в коробке','Сумма'];
  return book([[`${brand} ${city}`],header,[article,name,'50 мл',100,{t:'n',f:'D3*(1-30%)',v:70},quantity,3,{t:'n',f:'E3*F3',v:0}]], 'Бланк');
}
export function macroBook(bytes) {
  const files=unzipSync(new Uint8Array(bytes));
  // Opaque synthetic parts verify ZIP preservation, without committing real macros.
  files['xl/vbaProject.bin']=new Uint8Array([0,1,2,255,17,33]);
  files['customXml/parity-sentinel.xml']=strToU8('<sentinel unchanged="yes"/>');
  return zipSync(files);
}
