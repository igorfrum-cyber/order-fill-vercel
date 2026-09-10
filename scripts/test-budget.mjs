import assert from 'node:assert/strict';
import {christinaNorthSupplierQuantity,defaultNorthActualSupplierOrder} from '../src/workbookProcessor.js';
for(const [brand,position,need,expected] of [
 ['angiopharm',{blankBoxSize:6},4.75,6],['levissime',{blankBoxSize:4},4.75,8],
 ['skin_synergy',{},0.22,1],['sothys',{},1.25,2],['christina',{},4.75,6],
 ['klapp',{},10,9],['klapp',{},11,12],['novacutan',{novacutanMinimum:100},104,100],
 ['novacutan',{novacutanMinimum:100},105,110],['novacutan',{novacutanMinimum:10,supplierUnitSize:5},12,12],
]) assert.equal(defaultNorthActualSupplierOrder({brand},position,need),expected,brand);
for(const [need,quantity] of [[0,null],[0.22,3],[1.25,3],[2,3],[3,3],[4.75,6],[6,6]])assert.equal(christinaNorthSupplierQuantity(need),quantity);
import { planBudget, discountValue, coverage } from '../src/budgetPlanner.js';
import { priceOrderRows } from '../src/orderPricing.js';
import { budgetChangeComment } from '../src/budgetDialog.js';
assert.equal(budgetChangeComment({before:13,quantity:4,unit:1}),'Уменьшено на 9 шт. Для снижения заказа до указанной суммы.');
assert.equal(budgetChangeComment({before:3,quantity:0,unit:5}),'Уменьшено на 3 уп. Для снижения заказа до указанной суммы.');
assert.equal(budgetChangeComment({before:0,quantity:0,unit:1}),'');
assert.equal(budgetChangeComment({quantity:4,unit:1}),'');
assert.equal(budgetChangeComment({before:4,quantity:13,unit:1}),'Добавилось 9 шт. Для закупа до суммы.');
import { applyBudgetWorkbookPricing, budgetOrderRules, budgetReportRows, fillWorkbook, loadXlsx, buildNorthOrderFiles, finalizeNorthOrderFiles, saveXlsx } from '../src/workbookProcessor.js';
import { utils, write, read } from 'xlsx';
import { mkdirSync, writeFileSync } from 'node:fs';
const row = (key,category='A',extra={}) => ({key,name:key,category,quantity:10,demand:10,stock:0,transit:0,delivery:0.25,price:100,unit:1,step:1,minimum:1,...extra});
assert.equal(discountValue('30%'),30);
assert.equal(discountValue('30'),30);
const priceRows=[{group:'home',quantity:15,prices:[{id:'3',column:3,price:100,label:'Цена'}]},{group:'proff',quantity:10,prices:[{id:'4',column:4,price:80,label:'Цена со скидкой'}]}];
const fixed=priceOrderRows(priceRows,{home:{column:'3',discount:30},proff:{column:'4',discount:0}});
assert.equal(fixed[0].price,70);assert.equal(fixed[0].quantity,15);assert.equal(fixed[1].price,80);
assert.equal(priceOrderRows([{...priceRows[0],quantity:20}],{home:{column:'3',discount:30}})[0].quantity*70,1400);
assert.throws(()=>discountValue('100%'));
let p=planBudget([row('a'),row('c','C')],2200);
assert.equal(p.rows[0].quantity,10); assert.equal(p.rows[1].quantity,12);
p=planBudget([row('manual','A',{quantity:15})],1700);
assert.equal(p.rows[0].before,15); assert.equal(p.rows[0].quantity,17);
p=planBudget([row('locked','C',{locked:true}),row('free')],2200);
assert.equal(p.rows[0].quantity,10); assert.equal(p.rows[1].quantity,12);
p=planBudget([row('none','C',{demand:0}),row('free')],2200);
assert.equal(p.rows[0].quantity,10);
p=planBudget([row('zero','C',{quantity:0}),row('manual-zero','C',{quantity:0,excluded:true})],200);
assert.equal(p.rows[0].quantity,2); assert.equal(p.rows[1].quantity,0);
p=planBudget([row('cap','A',{quantity:60})],6200);
assert.equal(p.reason,'overSix');
p=planBudget([row('cap','A',{quantity:60})],6200,{allowOverSix:true}); assert.equal(p.total,6200);
p=planBudget([row('floor','A',{quantity:15})],1200); assert.equal(p.reason,'belowOne'); assert.equal(p.total,1300);
p=planBudget([row('floor','A',{quantity:15})],1200,{allowBelowOne:true}); assert.equal(p.total,1200);
p=planBudget([row('c','C',{quantity:10}),row('a','A',{quantity:10})],1800,{allowBelowOne:true});
assert.equal(p.rows[0].quantity,8); assert.equal(p.rows[1].quantity,10);
const nova=budgetOrderRules('novacutan','Novacutan SBIO, 2 мл');
assert.deepEqual(nova,{unit:1,step:10,minimum:100});
p=planBudget([row('nova','A',{...nova,quantity:0,demand:100})],9800); assert.equal(p.total,10000); assert.ok(p.complete);
p=planBudget([row('nova','A',{...nova,quantity:0,demand:100})],9000); assert.ok(!p.complete);
const mask=budgetOrderRules('novacutan','EYE FILLER MASK NOVACUTAN'); assert.equal(mask.unit,5);
for (const [brand,box,step] of [['christina',null,3],['klapp',null,3],['angiopharm',6,6],['levissime',4,4],['skin_synergy',null,1],['sothys',null,1]]) {
  const rules=budgetOrderRules(brand,'Товар',box);
  assert.equal(rules.step,step,brand);
  const planned=planBudget([row(brand,'A',{...rules,quantity:0,demand:100})],step*100);
  assert.equal(planned.rows[0].quantity,step,brand);
  assert.ok(planned.complete,brand);
}
assert.equal(coverage(row('mask','A',{...mask,demand:10}),12),6);
for(let i=1;i<80;i++) {
  const source=[row('a','A',{quantity:i,price:37}),row('b','B',{quantity:20,price:59})];
  const result=planBudget(source,3000,{allowOverSix:true,allowBelowOne:true});
  if(result.complete) assert.ok(result.total>=3000 && result.total<=3150 || result.before>=3000 && result.total<=3000);
  assert.deepEqual(source[0].quantity,i);
}

function book(rows,name='Тюмень') { const w=utils.book_new(); utils.book_append_sheet(w,utils.aoa_to_sheet(rows),name); return loadXlsx(write(w,{type:'buffer',bookType:'xlsx'})); }
const months=['сентябрь 2025','октябрь 2025','ноябрь 2025','декабрь 2025','январь 2026','февраль 2026','март 2026','апрель 2026','май 2026','июнь 2026','июль 2026','август 2026'];
const source=book([
 ['Тюмень','Период: 01.09.2025 - 31.08.2026','Прошлый период: 01.09.2025 - 30.11.2025'],
 ['', '',...months,'Итого'],
 ['Артикул','Товар',...months.map(()=>'Количество'),'Количество','Сумма выручки','% выручки','Кумулятивный %','Категория','Среднее за месяц','Количество прошлый период','Целевой запас','Остаток','В пути','Рекомендованный заказ','Заказано по факту','Комментарий'],
 ['P1','Крем 50 мл',...months.map(()=>10),120,1000,100,100,'C',10,30,12.5,0,0,13,'',''],
]);
const blank=()=>book([
 ['Бланк'], ['Артикул','Наименование','Цена','Цена со скидкой','Количество','Сумма'],
 ['P1','Крем 50 мл',100,{t:'n',f:'C3*(1-30%)',v:70},10,{t:'n',f:'D3*E3',v:700}]
], 'Бланк');
const result=fillWorkbook({sourceWorkbook:source,blankWorkbook:blank(),brand:'skin_synergy',orderMonth:'2026-10',blankId:'main'});
const data=budgetReportRows(result,'skin_synergy');
assert.equal(data[0].demand,10); assert.equal(data[0].category,'C');
assert.equal(data[0].prices[1].price,70);
// Uneven sales distinguish the existing business formula from a plain mean.
const unevenBook=read(saveXlsx(source),{type:'buffer'});
const unevenSheet=unevenBook.Sheets[unevenBook.SheetNames[0]];
const sales=[10,20,30,40,50,60,70,80,90,100,110,120];
sales.forEach((v,i)=>{unevenSheet[utils.encode_cell({r:3,c:i+2})]={t:'n',v};});
const unevenResult=fillWorkbook({sourceWorkbook:loadXlsx(write(unevenBook,{type:'buffer',bookType:'xlsx'})),blankWorkbook:blank(),brand:'skin_synergy',orderMonth:'2026-10',blankId:'main'});
const unevenMetric=budgetReportRows(unevenResult,'skin_synergy')[0];
const threshold=sales.reduce((a,b)=>a+b,0)/24;
const relevant=sales.filter(v=>v>threshold);
const businessDemand=relevant.reduce((a,b)=>a+b,0)/relevant.length;
assert.equal(unevenMetric.demand,businessDemand);
assert.notEqual(unevenMetric.demand,sales.reduce((a,b)=>a+b,0)/12);
const north=buildNorthOrderFiles([{workbook:blank(),fileName:'Тюмень.xlsx'},{workbook:blank(),fileName:'Сургут.xlsx'}],{brand:'skin_synergy',tyumenSourceWorkbook:source});
const christina=buildNorthOrderFiles([{workbook:blank(),fileName:'Тюмень.xlsx',variant:'home'},{workbook:blank(),fileName:'Сургут.xlsx',variant:'home'}],{brand:'christina',tyumenSourceWorkbook:source});
assert.equal(christina.planRows[0].actualSupplierOrder,21);
const christinaExport=finalizeNorthOrderFiles(christina);
assert.equal(read(saveXlsx(christinaExport.summaryWorkbook),{type:'buffer'}).Sheets['Бланк'].E3.v,21);
assert.equal(christinaExport.transfers[0].items[0].quantity,10);
assert.equal(north.planRows[0].budget.demand,10);
north.planRows[0].budgetComment='Добавилось 5 шт. Для закупа до суммы.';
const finalized=finalizeNorthOrderFiles(north,[{key:north.planRows[0].key,actualSupplierOrder:40}],{allowShortSupplierOrder:true});
assert.equal(finalized.transfers[0].items[0].quantity,10);
assert.match(finalized.orderTable.rows[0].comment,/Добавилось 5/);
const saved=read(saveXlsx(finalized.summaryWorkbook),{type:'buffer'});
assert.equal(saved.Sheets[saved.SheetNames[0]].D3.f,'C3*(1-30%)');
console.log('Budget stages, permissions, current edits, units, prices, formula preservation and north transfers: passed');
const pricingEntry={...data[0],quantity:10,price:80,pricing:{column:3,label:'Цена',basePrice:100,discount:20}};
for(let repeat=0;repeat<2;repeat++) {
  const priced=applyBudgetWorkbookPricing(blank(),'Бланк',2,[pricingEntry]);
  const exported=read(saveXlsx(priced),{type:'buffer'}).Sheets['Бланк'];
  assert.equal(exported.D3.v,80);
  assert.equal(exported.F3.v,800);
  assert.equal(exported.F3.f,'D3*E3');
  assert.match(exported.D3.f,/ROUND/);
}
const sharedDiscount=book([
 ['Скидка',0.3],['Артикул','Наименование','Цена','Цена со скидкой','Количество','Сумма'],
 ['P1','Крем 50 мл',100,{t:'n',f:'C3*(1-$B$1)',v:70},10,{t:'n',f:'D3*E3',v:700}],
 ['', '', '', '', '', {t:'n',f:'SUM(F3:F3)',v:700}],
], 'Бланк');
const direct=read(saveXlsx(applyBudgetWorkbookPricing(sharedDiscount,'Бланк',2,[pricingEntry])),{type:'buffer'}).Sheets['Бланк'];
assert.equal(direct.B1.v,0.2); assert.equal(direct.D3.f,'C3*(1-$B$1)');
assert.equal(direct.F4.v,800);
assert.equal(read(saveXlsx(sharedDiscount),{type:'buffer'}).Sheets['Бланк'].B1.v,0.3);
const net=read(saveXlsx(applyBudgetWorkbookPricing(sharedDiscount,'Бланк',2,[{...pricingEntry,price:70,pricing:{column:4,basePrice:70,discount:0}}])),{type:'buffer'}).Sheets['Бланк'];
assert.equal(net.D3.v,70); assert.equal(net.B1.v,0.3);
north.budgetPricing=[{...pricingEntry,key:north.planRows[0].key,group:'main'}];
const pricedNorth=finalizeNorthOrderFiles(north,[{key:north.planRows[0].key,actualSupplierOrder:40}],{allowShortSupplierOrder:true});
const northSheet=read(saveXlsx(pricedNorth.summaryWorkbook),{type:'buffer'}).Sheets['Бланк'];
assert.equal(northSheet.F3.v,3200);
assert.equal(pricedNorth.transfers[0].items[0].quantity,10);
console.log('Export discount, formula caches, net pricing, original preservation and north pricing: passed');
mkdirSync('test-output/budget', { recursive:true });
writeFileSync('test-output/budget/Тюмень таблица.xlsx',saveXlsx(source));
writeFileSync('test-output/budget/Тюмень.xlsx',saveXlsx(blank()));
writeFileSync('test-output/budget/Сургут.xlsx',saveXlsx(blank()));
