import { budgetReportRows } from "./../../workflows/budget/adapters.js";
import { priceOrderRows } from "./../../domain/budget/pricing.js";
import { money } from "./../../shared/money.js";
import { procurementTotalCents } from "./../../domain/budget/christina-lines.js";
import { budgetChangeComment, openBudgetDialog } from "./dialog.js";
import { createOrderPricing } from "./../pricing/settings.js";
import { budgetOrderRules } from "./../../domain/order/budget-rules.js";
import { applyBudgetWorkbookPricing } from "./../../infrastructure/excel/pricing.js";
import { budgetNorthWorkbookPricing } from "./../../workflows/north/export.js";
import { workbookPriceOptions } from "./../../infrastructure/excel/blanks.js";

export function createBudgetController(context) {
  const {elements, state, actions} = context;
function liveBudgetRows(north) {
  if(!north)return state.currentResults.flatMap(r=>budgetReportRows(r,actions.selectedBrand())).map(r=>({...r,quantity:Number(state.editState.get(r.key)?.value||0)}));
  return (state.currentNorthResult?.planRows||[]).map(r=>{
    const tr=[...elements.northPlanBody.querySelectorAll('tr[data-key]')].find(el=>el.dataset.key===r.key);
    return {...r.budget,key:r.key,name:r.name,group:r.variant||'main',quantity:Number(tr?.querySelector('.north-actual-input').value||0)};
  });
}

function refreshBudgetTotals() {
  for(const north of [false,true]) {
    const panel=state.budgetPanels.get(north); if(!panel)continue;
    const rows=liveBudgetRows(north);
    const settings=north?state.currentNorthResult?.priceSettings:state.currentResults[0]?.priceSettings;
    if(!settings)continue;
    const priced=priceOrderRows(rows,settings);
    const groups=[...new Set(priced.map(r=>r.group))];
    const host=panel.querySelector('[data-budget-totals]');
    for(const g of groups) {
      let section=[...host.children].find(e=>e.dataset.group===g);
      if(!section){section=document.createElement('div');section.dataset.group=g;section.innerHTML='<strong></strong><label>Желаемая сумма, ₽<input type="number" min="0" step="0.01" data-budget-target></label>';host.append(section);}
      const selected=priced.filter(r=>r.group===g);
      section.querySelector('strong').textContent=`${g==='main'?'Текущая сумма':g.toUpperCase()}: ${selected.some(r=>r.quantity>0&&!r.price)?'Не определена цена':money(procurementTotalCents(selected)/100)+' ₽'}`;
    }
    for(const child of [...host.children])if(!groups.includes(child.dataset.group))child.remove();
  }
}

function budgetComment(previous, row) {
  const note = budgetChangeComment(row);
  // The report uses a single-line input, which strips newline separators.
  return [previous, note].filter(Boolean).join('; ');
}

function installBudgetControls() {
  for (const north of [false, true]) {
    const anchor = north ? elements.northDownloadButton : elements.downloadButton;
    const button = document.createElement('button');
    const undo = north ? elements.northBudgetUndo : elements.budgetUndo;
    button.type = undo.type = 'button';
    button.className = undo.className = 'secondary';
    button.textContent = 'Заказ до суммы';
    undo.textContent = 'Отменить перерасчет суммы';
    undo.hidden = true;
    const panel=document.createElement('section');panel.className='budget-panel';
    panel.innerHTML='<h3>Заказ до суммы</h3><div data-budget-totals></div><div class="budget-actions"></div>';
    (north?elements.northResult:elements.resultEl).querySelector('.result-head').after(panel);
    panel.querySelector('.budget-actions').append(button,undo);state.budgetPanels.set(north,panel);
    button.onclick = () => {
      if (north && !state.currentNorthResult) return alert('Сначала соедините бланки.');
      if (!north && !state.currentResults.length) return alert('Сначала заполните бланк.');
      if (!north && actions.selectedBrand() !== 'angiopharm' && state.currentResults.some(r => r.summary.sourceCity !== 'Тюмень')) return alert('Заказ до суммы доступен только для Тюмени. Для северных городов используйте раздел «Север».');
      if (north && !state.currentNorthResult.hasTyumenSource) return alert('Загрузите таблицу Тюмени для учета продаж и остатков.');
      const brand = north ? actions.selectedNorthBrand() : actions.selectedBrand();
      const locks = north ? state.northBudgetLocks : state.budgetLocks;
      const rows = north ? state.currentNorthResult.planRows.map(r => {
        const tr = [...elements.northPlanBody.querySelectorAll('tr[data-key]')].find(el => el.dataset.key === r.key);
        const calculated = actions.recalculateNorthRow(r,actions.northCityQuantities(tr));
        return { ...r.budget, ...budgetOrderRules(brand,r.name,r.novacutanMinimum ?? r.blankBoxSize),
          key:r.key, name:r.name, group:r.variant || 'main', comment:r.budgetComment || '',
          inventoryKey:r.baseKey || r.key,
          stock:r.tyumenStock, transit:r.tyumenInTransit, outbound:calculated.northNeed,
          quantity:Number(tr.querySelector('.north-actual-input').value || 0), locked:locks.has(r.key),
        };
      }) : state.currentResults.flatMap(result => budgetReportRows(result,brand)).map(r=>({
        ...r, quantity:Number(state.editState.get(r.key)?.value || 0), comment:state.editState.get(r.key)?.comment || '', locked:locks.has(r.key),
      }));
      const counts = new Map();
      for (const r of rows) counts.set(r.inventoryKey,(counts.get(r.inventoryKey)||0)+1);
      for (const r of rows) if (counts.get(r.inventoryKey)>1) r.unsafe=true;
      const settings=north?state.currentNorthResult.priceSettings:state.currentResults[0].priceSettings;
      const targets=Object.fromEntries([...panel.querySelector('[data-budget-totals]').children].map(el=>[el.dataset.group,el.querySelector('input').value]));
      openBudgetDialog({ rows:priceOrderRows(rows,settings), fixedPricing:true, initialTargets:targets, christina:brand==='christina', apply:(planned,newLocks)=>{
        // Validate export pricing before changing any manager edits.
        if (north) {
          for (const summary of state.currentNorthResult.summaries?.length ? state.currentNorthResult.summaries : [state.currentNorthResult.summary]) budgetNorthWorkbookPricing(summary, planned);
        } else {
          for (const result of state.currentResults) applyBudgetWorkbookPricing(result.blankWorkbook,result.blankDetection.sheetName,result.blankDetection.headerRow,planned.filter(r=>r.group===result.blankId));
        }
        const pricingOwners = north ? [state.currentNorthResult] : state.currentResults;
        const oldPricing = pricingOwners.map(owner=>owner.budgetPricing);
        for (const owner of pricingOwners) owner.budgetPricing = north ? planned : planned.filter(r=>r.group===owner.blankId);
        // Keep the pre-apply edits, not the initial recommendations, for undo.
        const oldLocks = new Set(locks);
        const oldEdits = new Map([...state.editState].map(([k,v])=>[k,{...v}]));
        const oldNorth = north ? state.currentNorthResult.planRows.map(r=>({key:r.key,comment:r.budgetComment})) : [];
        const oldNorthValues = north ? actions.collectNorthPlanEdits() : [];
        locks.clear(); for(const key of newLocks) locks.add(key);
        for(const r of planned) {
          if(north) {
            // Supplier quantity only: never rewrite the city inputs here.
            const source=state.currentNorthResult.planRows.find(p=>p.key===r.key);
            source.budgetComment=budgetComment(source.budgetComment,r);
            const tr=[...elements.northPlanBody.querySelectorAll('tr[data-key]')].find(el=>el.dataset.key===r.key);
            const input=tr.querySelector('.north-actual-input');
            input.value=r.quantity; input.dataset.manual='true';
            tr.querySelector('[data-north-budget-lock]').checked=locks.has(r.key);
            state.northPlanEdits.set(r.key,r.quantity); actions.updateNorthRowDisplay(tr);
          } else {
            const edit=state.editState.get(r.key);
            state.editState.set(r.key,{...edit,value:r.quantity,comment:budgetComment(edit.comment,r)});
          }
        }
        if(north) actions.clearNorthDownloadLinks(); else { actions.clearDownloadLinks(); actions.renderReportView(); }
        undo.hidden=false;
        refreshBudgetTotals();
        undo.onclick=()=>{
          pricingOwners.forEach((owner,i)=>{owner.budgetPricing=oldPricing[i];});
          locks.clear(); for(const key of oldLocks) locks.add(key);
          if(north) {
            for(const old of oldNorth) state.currentNorthResult.planRows.find(r=>r.key===old.key).budgetComment=old.comment;
            for(const old of oldNorthValues) {
              const tr=[...elements.northPlanBody.querySelectorAll('tr[data-key]')].find(el=>el.dataset.key===old.key);
              tr.querySelector('.north-actual-input').value=old.actualSupplierOrder ?? '';
              tr.querySelector('[data-north-budget-lock]').checked=locks.has(old.key);
              actions.updateNorthRowDisplay(tr);
            }
            actions.clearNorthDownloadLinks();
          } else { state.editState=oldEdits; actions.renderReportView(); actions.clearDownloadLinks(); }
          undo.hidden=true;
          refreshBudgetTotals();
        };
      }});
    };
  }
  const ordinaryFiles=()=>actions.selectedBrand()==='christina'?[{group:'home',file:elements.homeFile.files[0]},{group:'proff',file:elements.proffFile.files[0]}].filter(e=>e.file):elements.blankFile.files[0]?[{group:'main',file:elements.blankFile.files[0]}]:[];
  state.orderPricing=createOrderPricing(elements.form,ordinaryFiles,f=>actions.loadWorkbook(f,{allowLegacyXls:actions.selectedBrand()==='novacutan'}),(w,n)=>workbookPriceOptions(w,n,actions.selectedBrand()),actions.resetFillState);
  state.northPricing=createOrderPricing(elements.northForm,()=>actions.northFilesForMerge().map(e=>({...e,group:e.variant||'main'})),actions.loadWorkbook,(w,n)=>workbookPriceOptions(w,n,actions.selectedNorthBrand()),()=>{elements.northResult.classList.add('hidden');state.currentNorthResult=null;actions.clearNorthDownloadLinks();});
  for(const input of [elements.blankFile,elements.homeFile,elements.proffFile,elements.brandSelect])input.addEventListener('change',()=>state.orderPricing.refresh());
  for(const input of [elements.northFileInput,elements.northHomeInput,elements.northProffInput,elements.northBrandSelect])input.addEventListener('change',()=>state.northPricing.refresh());
  for(const list of [elements.northFileList,elements.northHomeFileList,elements.northProffFileList])list.addEventListener('click',()=>state.northPricing.refresh());
}
  return { liveBudgetRows, refreshBudgetTotals, budgetComment, installBudgetControls };
}
