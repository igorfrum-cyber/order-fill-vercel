import { model } from './refactor-analysis.mjs';
import { execFileSync } from 'node:child_process';
import { writeFile, readFile, mkdir } from 'node:fs/promises';
import path from 'node:path';
const text=execFileSync('git',['show','1e2baa1354c40156ff8f559ad5b2f2d27e119a0e:src/app.js'],{encoding:'utf8'});
const {ast,bindings}=model(text);
const mapping=JSON.parse(await readFile('docs/plans/export-owners.json','utf8')).owners;
const groups=new Map();
function assign(owner,names){names.split(/\s+/).filter(Boolean).forEach(name=>groups.set(name,owner));}
assign('ui/budget/controller.js','liveBudgetRows refreshBudgetTotals budgetComment installBudgetControls');
assign('app/lifecycle.js','setDefaultOrderMonth setActiveMode bindFileName selectedBrand adjustmentLabelForBrand mainBlankLabelForBrand selectedNorthBrand configureNorthBrandUploads configureBrandFields setSubmitButtonState resetFillState scrollTargetForKeyboard');
assign('ui/north/uploads.js','resetNorthCalculationState renderNorthFileList sameNorthFile pendingNorthFiles uniqueNorthFiles northFilesForMerge addPendingNorthFiles removeNorthFile handleNorthFileListClick isNorthChristinaMode resetNorthUploadedFiles');
assign('ui/order/report.js','statusLabel escapeHtml combinedSummary renderMetrics initialComment rowEdit isIssueRow isPriorityRow rowMatchesFilter rowSearchText rowMatchesSearch filterTitle duplicateDescription duplicateDetailsHtml baselineForReportRow renderRows renderReportView handleReportInput');
assign('ui/order/issues.js','missingInBlankRows duplicateSignature sourceDuplicateRows issueReportRows issueReason csvCell issueReportCsv');
assign('ui/order/edits.js','collectEdits validateEdits isManualDeviation confirmQualityWarnings rowNeedsComment updateCommentHint');
assign('ui/files/loader.js','loadWorkbook assertDifferentTyumenFiles isLegacyXls normalizeWorkbookBytes');
assign('ui/downloads/browser.js','clearDownloadLinks clearNorthDownloadLinks triggerDownload downloadIssueReport prepareDownloadLinks prepareNorthDownloadLinks');
assign('infrastructure/excel/transfer-tables.js','todayRu transferWorkbookBytes northOrderTableBytes');
assign('ui/north/plan.js','formatNorthQuantity formatNorthCommentQuantity supplierUnitsFromPieces demandPiecesFromSupplierUnits northTransferParts northSupplierOrderText supplierPartsForNorthActual northPlanComment nearestNorthMultiple northCityInputs northStockText renderNorthPlan defaultNorthActual northCityQuantities recalculateNorthRow updateNorthRowDisplay collectNorthPlanEdits northShortOrderWarnings confirmNorthShortOrders confirmNorthMerge');
const imported=new Map();
const moved={openBudgetDialog:'ui/budget/dialog.js',budgetChangeComment:'ui/budget/dialog.js',createOrderPricing:'ui/pricing/settings.js',priceOrderRows:'domain/budget/pricing.js',money:'shared/money.js',procurementTotalCents:'domain/budget/christina-lines.js',installTableRecalculation:'ui/recalculation/controller.js'};
for(const n of ast.body.filter(n=>n.type==='ImportDeclaration'))for(const s of n.specifiers)imported.set(s.local.name,{name:s.imported?.name || s.local.name,path:n.source.value==='./workbookProcessor.js'?mapping[s.local.name]:moved[s.local.name] || n.source.value});
const kinds=new Map();
for(const n of ast.body.filter(n=>n.type==='VariableDeclaration'))for(const d of n.declarations){
  const name=d.id.name;
  kinds.set(name, ['NORTH_CITIES','NORTH_ALLOCATION_ORDER','NORTH_TRANSFER_DISPLAY_ORDER'].includes(name)?'constant':text.slice(d.init?.start||0,d.init?.end||0).includes('document.') || name==='submitButton' || name==='northSubmitButton' ?'elements':'state');
}
const parents=new Map();function walk(n){if(!n||typeof n!=='object')return;for(const [k,v] of Object.entries(n)){if(k==='range')continue;for(const c of Array.isArray(v)?v:[v])if(c?.type){parents.set(c,n);walk(c);}}}walk(ast);
const refs=[...bindings].flatMap(([name,v])=>v.references.map(r=>({name,id:r.identifier})));
const modules=new Map();const used=new Map();
function render(node,owner){
  const edits=[];if(!used.has(owner))used.set(owner,new Set());
  for(const {name,id} of refs.filter(r=>r.id.start>=node.start&&r.id.end<=node.end)){
    let replacement;
    if(imported.has(name)||kinds.get(name)==='constant'){used.get(owner).add(name);continue;}
    if(kinds.has(name))replacement=`${kinds.get(name)}.${name}`;
    else if(groups.has(name)){if(groups.get(name)===owner)continue;replacement=`actions.${name}`;}
    else throw Error(`Unknown app binding ${name}`);
    const parent=parents.get(id);if(parent?.type==='Property'&&parent.shorthand&&parent.value===id)replacement=`${name}: ${replacement}`;
    edits.push({start:id.start-node.start,end:id.end-node.start,replacement});
  }
  let body=text.slice(node.start,node.end);for(const e of [...new Map(edits.map(e=>[e.start,e])).values()].sort((a,b)=>b.start-a.start))body=body.slice(0,e.start)+e.replacement+body.slice(e.end);
  return body;
}
function append(owner,body){if(!modules.has(owner))modules.set(owner,[]);modules.get(owner).push(body);}
const bootstrap=[];const factories=[];const constants=[];
for(const n of ast.body){
  if(n.type==='ImportDeclaration')continue;
  if(n.type==='FunctionDeclaration'){
    const owner=groups.get(n.id.name);if(!owner)throw Error(`Missing app owner ${n.id.name}`);
    append(owner,render(n,owner));continue;
  }
  if(n.type==='VariableDeclaration'){
    for(const d of n.declarations){if(kinds.get(d.id.name)==='constant'){constants.push(`export const ${text.slice(d.start,d.end)};`);continue;}
      bootstrap.push(`${kinds.get(d.id.name)}.${d.id.name} = ${d.init?render(d.init,'app/bootstrap.js'):'undefined'};`);}
    continue;
  }
  const original=text.slice(n.start,n.end);
  if(original.includes('.addEventListener(')&&original.split('\n').length>5&&n.type==='ExpressionStatement'){
    const target=n.expression.callee.object.name,event=n.expression.arguments[0].value;
    const owner=target?.startsWith('north')?'ui/north/events.js':target?.startsWith('warehouse')?'ui/tyumen/events.js':'ui/order/events.js';
    const name=`bind${target[0].toUpperCase()+target.slice(1)}${event[0].toUpperCase()+event.slice(1)}`;
    append(owner,`export function ${name}(context) {\n  const {elements, state, actions} = context;\n${render(n,owner)}\n}`);
    factories.push({name,owner,event:true});bootstrap.push(`${name}(context);`);
  }else bootstrap.push(render(n,'app/bootstrap.js'));
}
function imports(owner){
  const byPath=new Map();for(const name of used.get(owner)||[]){const imp=imported.get(name)||{name,path:'ui/north/display-cities.js'};const p=imp.path.startsWith('.')?imp.path:imp.path.includes('/')&&imp.path.endsWith('.js')?`./${path.posix.relative(path.posix.dirname(owner),imp.path)}`:imp.path;
    if(!byPath.has(p))byPath.set(p,[]);byPath.get(p).push(imp.name===name?name:`${imp.name} as ${name}`);}
  return [...byPath].map(([p,names])=>`import { ${[...new Set(names)].sort().join(', ')} } from ${JSON.stringify(p)};`).join('\n');
}
for(const [owner,bodies] of modules){
  await mkdir(`src/${path.posix.dirname(owner)}`,{recursive:true});let body;
  if(owner.endsWith('/events.js'))body=bodies.join('\n\n');
  else if(owner==='infrastructure/excel/transfer-tables.js'){
    // These builders depend only on plain transfer data, dates and SheetJS.
    body=bodies.map(b=>`export ${b}`).join('\n\n');
    for(const name of groups.keys())if(groups.get(name)===owner)factories.push({name,owner,direct:true});
  }else{
    const names=[...groups].filter(([,o])=>o===owner).map(([name])=>name);const name=`create${owner.split('/').slice(-2,-1)[0][0].toUpperCase()+owner.split('/').slice(-2,-1)[0].slice(1)}${path.basename(owner,'.js').replace(/(^|-)(\w)/g,(_,a,b)=>b.toUpperCase())}`;
    factories.push({name,owner});body=`export function ${name}(context) {\n  const {elements, state, actions} = context;\n${bodies.join('\n\n')}\n  return { ${names.join(', ')} };\n}`;
  }
  await writeFile(`src/${owner}`,imports(owner)+'\n\n'+body+'\n');
}
await writeFile('src/ui/north/display-cities.js',constants.join('\n\n')+'\n');
const factoryImports=factories.map(f=>`import { ${f.name} } from './${path.posix.relative('app',f.owner)}';`).join('\n');
const initialize=factories.filter(f=>!f.event&&!f.direct).map(f=>`Object.assign(actions, ${f.name}(context));`).join('\n');
const direct=factories.filter(f=>f.direct).map(f=>`${f.name}: ${f.name}`).join(', ');
await writeFile('src/app/bootstrap.js',imports('app/bootstrap.js')+'\n'+factoryImports+`\n\nexport function startApplication() {\nconst elements = {}, state = {}, actions = {${direct}};\nconst context = {elements, state, actions};\n${initialize}\n${bootstrap.join('\n\n')}\n}\n`);
await writeFile('src/app.js',"import './styles.css';\nimport {startApplication} from './app/bootstrap.js';\n\nstartApplication();\n");
