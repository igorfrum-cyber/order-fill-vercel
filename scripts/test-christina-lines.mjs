import assert from 'node:assert/strict';
import { readFileSync, existsSync } from 'node:fs';
import { christinaLineGroups, procurementTotalCents, proposeLineStep, completeChristinaLines } from '../src/christinaLines.js';
import { planBudget } from '../src/budgetPlanner.js';
import { priceOrderRows } from '../src/orderPricing.js';
import { loadXlsx, saveXlsx, christinaProffLines, applyBudgetWorkbookPricing, fillWorkbook, budgetReportRows,
  buildNorthOrderFiles, finalizeNorthOrderFiles } from '../src/workbookProcessor.js';

const fixture = (quantities = [18,18,18,3], id = 'MUSE') => quantities.map((quantity, i) => ({
  key: `${id}:${i}`, name: `${id} ${i}`, group: 'proff', category: 'A+', price: 100,
  quantity, demand: 10, stock: 0, transit: 0, unit: 1, step: 3, minimum: 3, delivery: .25,
  line: { id, name: id, article: String(i), required: quantities.map((_, j) => String(j)) },
}));
const base = rows => new Map(christinaLineGroups(rows).map(l => [l.id, l.netCents]));
const propose = (rows, target = 10000) => proposeLineStep(rows, 'MUSE', base(rows), target);
const rows = fixture();
assert.equal(procurementTotalCents(rows), 564000);
assert.equal(christinaLineGroups(rows)[0].sets, 3);
assert.equal(procurementTotalCents(rows.map(r => ({ ...r, group: 'home' }))), 570000);
assert.equal(christinaLineGroups(rows.slice(1))[0].sets, 0, 'Missing membership cannot earn a discount');
assert.equal(christinaLineGroups([...rows, rows[0]])[0].sets, 0, 'Duplicate membership cannot earn a discount');
assert.equal(christinaLineGroups(rows.map((r,i) => i ? r : {...r, unsafe:true}))[0].sets, 0);
const step = propose(rows);
assert(step.accepted);
assert.deepEqual(step.rows.map(r => r.quantity), [18,18,18,6]);
assert.equal(step.addedCents, 30000);
assert.equal(step.savedCents, 6000);
assert.equal(step.totalCents, 588000);
assert.deepEqual(rows.map(r=>r.quantity), [18,18,18,3], 'Pure preview');
assert.equal(propose(fixture([18,18,3,3])).reason, 'saving');
assert.equal(propose(rows.map(r=>({...r,locked:true}))).reason, 'locked');
assert.equal(propose(rows.map(r=>({...r,stock:100}))).reason, 'coverage');
assert.equal(propose(rows, 5500).reason, 'targetReached');
assert.equal(propose(fixture([3,3,3,0]), 700).reason, 'targetLimit');
const exception = fixture([18,18,18,0]).map((r,i)=>i===3?{...r,demand:0,stock:1000}:r);
assert(propose(exception,5400).accepted, 'First three sets: one missing item may have no sales');
const afterException = propose(exception,5400).rows;
assert.equal(propose(afterException).reason, 'coverage', 'Exception cannot repeat for next sets');
assert.equal(propose(exception.map((r,i)=>i===3?{...r,excluded:true}:r)).reason, 'locked');
const completed = completeChristinaLines(rows, 10000);
assert(completed.steps.length > 0);
assert(christinaLineGroups(completed.rows)[0].netCents <= base(rows).get('MUSE')*1.3);
assert(completed.rejected.some(r=>r.reason==='lineGrowth'), '30% baseline is not reset per step');
const priced = priceOrderRows(rows.map(r=>({...r, prices:[{id:'4',column:4,label:'Клубная цена',price:100}]})), {proff:{column:'4',discount:30}});
assert.equal(procurementTotalCents(priced), 394800, 'Main 30%, then extra 5% only on three of each');
const result = planBudget(rows, 6100);
assert(result.complete);
assert(result.total >= 6100 && result.total <= 6405);
assert.deepEqual(result.rows.map(r=>r.before), [18,18,18,3]);
assert.equal(result.total*100, procurementTotalCents(result.rows));
const down = planBudget(fixture([18,18,18,18]), 5000, {allowBelowOne:true});
assert(down.complete);
assert(down.total <= 5000);
assert.equal(Math.round(down.total*100), procurementTotalCents(down.rows), 'Reduction revalues every complete set');
const locked = fixture().map(r=>({...r,locked:true}));
assert.equal(planBudget(locked,1000,{allowBelowOne:true}).complete,false);
assert.deepEqual(planBudget(locked,1000,{allowBelowOne:true}).rows.map(r=>r.quantity), [18,18,18,3]);

// Optional real-blank smoke tests; portable arithmetic tests above always run.
const path='/Users/igorfrumes/Downloads/Актуальный_бланк_PROFF от 01.09.26.xlsx';
if (existsSync(path)) {
  const workbook=loadXlsx(readFileSync(path));
  const lines=christinaProffLines(workbook,workbook.sheets[0]);
  assert.deepEqual([...new Map([...lines.values()].map(l=>[l.id,l.required.length]))], [
    ['MUSE',10],['NUANCE',10],['LINE REPAIR',7],['ILLUSTRIOUS',8],['BIO PHYTO',16],
    ['COMODEX',9],['FOREVER YOUNG',11],['UNSTRESS',10],['ROSE DE MER',6],
  ]);
  assert.equal(lines.has(115), false);
  const before=saveXlsx(workbook);
  const exported=applyBudgetWorkbookPricing(workbook,'Лист1',7,[{
    name:'Muse',quantity:3,price:1764,preserveBlankPrices:true,blankRow:11,quantityCol:5,
    pricing:{column:4,basePrice:2520,discount:30},
  }]);
  assert.deepEqual(saveXlsx(exported),before,'Club prices and formulas unchanged');
  const sourcePath='test-output/budget/Тюмень таблица.xlsx';
  if(existsSync(sourcePath)) {
    const source=loadXlsx(readFileSync(sourcePath));
    const filled=fillWorkbook({sourceWorkbook:source,sourceFileName:'Тюмень таблица.xlsx',blankWorkbook:loadXlsx(readFileSync(path)),brand:'christina',blankId:'proff',orderMonth:'2026-10'});
    const budget=budgetReportRows(filled,'christina');
    assert.equal(budget.filter(r=>r.line).length,87, 'Unmatched professional goods available for first-set exception');
    const north=buildNorthOrderFiles([{workbook:loadXlsx(readFileSync(path)),fileName:'Тюмень PROFF.xlsx',variant:'proff'}],{brand:'christina',tyumenSourceWorkbook:source,tyumenSourceFileName:'Тюмень таблица.xlsx'});
    assert.equal(north.planRows.filter(r=>r.budget.line).length,87);
    assert(north.planRows.every(r=>r.budget.preserveBlankPrices));
    const initialTransfers=JSON.stringify(north.transfers);
    const entries=priceOrderRows(north.planRows.map(r=>({...r.budget,group:'proff',key:r.key,quantity:3})),{proff:{column:'4',discount:30}});
    north.budgetPricing=entries;
    const final=finalizeNorthOrderFiles(north,entries.map(r=>({key:r.key,actualSupplierOrder:3})),{allowShortSupplierOrder:true});
    assert.equal(JSON.stringify(north.transfers),initialTransfers,'Supplier additions never mutate transfers');
    assert.equal(final.summaryWorkbook.sheets[0].cells.get('11:4').value,2520);
  }
}
console.log('CHRISTINA: sets, sequential discounts, limits, exceptions, locks, reduction and club-price export passed');
