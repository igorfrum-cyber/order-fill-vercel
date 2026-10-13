import { book, source, blank, macroBook } from '../fixtures/workbooks.mjs';
import { unzipSync } from 'fflate';

function parts(bytes) {
  return Object.fromEntries(Object.entries(unzipSync(new Uint8Array(bytes))).sort(([a],[b])=>a.localeCompare(b)).map(([name,value])=>[name,Buffer.from(value).toString('base64')]));
}
export async function scenarios(api, {planBudget, priceOrderRows}) {
  const data={exports:Object.keys(api).sort(), cases:{}};
  const put=(name, value)=>{data.cases[name]=value;};
  const save=w=>parts(api.saveXlsx(w));
  put('pure', {
    articles:['МТ4775',' MT-42 ','SK033'].map(v=>api.normalizeArticle(v)),
    rounding:[0,1.49,1.5,9.4,9.5,13].map(api.roundHalfUp),
    quantities:['angiopharm','christina','levissime','sothys','skin_synergy','klapp','novacutan'].map(brand=>[brand,[1,4.75,8,13,105].map(q=>api.adjustQuantityForBrand(q,brand,6))]),
    filenames:[api.outputFileName('input.xlsm','Тюмень'),api.sourceOutputFileName('input.xlsx')],
  });
  for (const brand of ['skin_synergy','angiopharm','levissime','christina','klapp']) {
    const result=api.fillWorkbook({sourceWorkbook:api.loadXlsx(source()),blankWorkbook:api.loadXlsx(blank(brand)),brand,orderMonth:'2026-10',blankId:brand==='christina'?'home':'main'});
    put(`order-${brand}`,{report:result.reportRows,summary:result.summary,blank:save(result.blankWorkbook),source:save(result.sourceWorkbook)});
    if(brand==='skin_synergy'){
      const edits=[{key:result.reportRows[0].key,blankRow:3,sourceRow:4,value:20,comment:'Ручной комментарий'}];
      const edited=api.applyFinalEdits({blankWorkbook:result.blankWorkbook,sourceWorkbook:result.sourceWorkbook,reportRows:result.reportRows,edits,brand});
      put('manual-order',{blank:save(edited.blankWorkbook),source:save(edited.sourceWorkbook)});
      put('stocks',save(api.addOrderBlankStocks(edited.blankWorkbook,result.blankDetection,result.reportRows,brand)));
    }
  }
  const office=source([{article:'P1',name:'Крем 50 мл',stock:10},{article:'P2',name:'Гель 100 мл',stock:0}]);
  const warehouse=source([{article:'P1',name:'Крем 50 мл',stock:90,sales:20},{article:'P3',name:'Только склад',stock:5,sales:0}]);
  for(const inventory of [false,true]) {
    const merged=api.mergeTyumenSources({officeWorkbook:api.loadXlsx(office),warehouseWorkbook:api.loadXlsx(inventory?source([{article:'P1',name:'Крем 50 мл',stock:90,sales:0}],{inventory:true}):warehouse),brand:'skin_synergy'});
    put(`tyumen-${inventory}`,save(merged));
  }
  const recalculated=api.recalculateOrderTable({workbook:api.loadXlsx(source([{article:'4775',name:'Lev Бальзам / HYDRAGE GLOBAL be.STICK 18,5 г',sales:10,stock:3,fact:0,comment:'Сохранить ноль'},{article:'МТ4775',name:'ЧЗ Lev Ночная маска/HYDRAGE GLOBAL be.STICK 18,5 г',sales:0,stock:30}])),brand:'levissime',orderMonth:'2026-10'});
  put('recalculation',{workbook:save(recalculated.workbook),rows:recalculated.rows});
  const north=api.buildNorthOrderFiles([{workbook:api.loadXlsx(blank('skin_synergy','Сургут',10)),fileName:'Сургут.xlsx'},{workbook:api.loadXlsx(blank('skin_synergy','Тюмень',5)),fileName:'Тюмень.xlsx'}],{brand:'skin_synergy',tyumenSourceWorkbook:api.loadXlsx(office)});
  const final=api.finalizeNorthOrderFiles(north,[{key:north.planRows[0].key,actualSupplierOrder:30}],{allowShortSupplierOrder:true});
  put('north',{plan:north.planRows,summary:save(final.summaryWorkbook),transfers:final.transfers,orderTable:final.orderTable});
  const row=(key,category,extra={})=>({key,name:key,category,quantity:10,demand:10,stock:0,transit:0,delivery:0.25,price:100,unit:1,step:1,minimum:1,...extra});
  put('budget', [planBudget([row('a','A'),row('c','C')],2200),planBudget([row('a','A'),row('c','C')],1200,{allowBelowOne:true}),planBudget([row('locked','A',{locked:true})],500),planBudget([row('cap','A',{quantity:60})],6200)]);
  put('pricing',priceOrderRows([{key:'p',group:'home',quantity:15,prices:[{id:'3',column:3,price:100,label:'Цена'}]}],{home:{column:'3',discount:30}}));
  put('macro-roundtrip',save(api.loadXlsx(macroBook(blank()))));
  return JSON.parse(JSON.stringify(data));
}
