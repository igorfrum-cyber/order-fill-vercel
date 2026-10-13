import { parseAst } from 'rollup/parseAst';
import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { createHash } from 'node:crypto';
const {analyze}=await import(process.env.SCOPE_MODULE || 'eslint-scope');
export function model(text) {
  const ast=parseAst(text);
  function walk(n){if(!n||typeof n!=='object')return;if(n.type)n.range=[n.start,n.end];for(const [k,v] of Object.entries(n))if(k!=='range'){if(Array.isArray(v))v.forEach(walk);else if(v?.type)walk(v);}}
  walk(ast);
  const manager=analyze(ast,{ecmaVersion:2024,sourceType:'module',optimistic:true,ignoreEval:true});
  const scope=manager.scopes.find(s=>s.type==='module');
  const bindings=new Map(scope.variables.map(v=>[v.name,v]));
  return {ast,manager,bindings};
}
if (process.argv[1]?.endsWith('refactor-analysis.mjs')) {
const source=execFileSync('git',['show','1e2baa1354c40156ff8f559ad5b2f2d27e119a0e:src/workbookProcessor.js'],{encoding:'utf8'});
const {ast,bindings}=model(source);
const groups={};
function assign(owner,names){for(const name of names.split(/\s+/).filter(Boolean))groups[name]=owner;}
assign('shared/text.js','asText normalizeHeader parseNumber');
assign('domain/matching/identity.js','ARTICLE_TRANSLATION normalizeArticle normalizeName similarity extractVolumeKeys volumeAwareSimilarity chooseCandidate chooseNameFallback chooseSothysNameFallback isCombinedChzName isChzCloneName comparableChzName chzNameSimilarity canonicalProductArticle levissimeRussianIdentity levissimeVolumeSignature levissimeEnglishIdentity matchingLevissimeEnglishNames');
assign('domain/brands/rules.js','BRAND_RULES brandRule articleNormalizeOptions blankDetectionOptions');
assign('domain/calculation/formulas.js','roundHalfUp normalizeCategory categoryCoefficient categoryFromCumulative calculateTargetNew positiveSale newProductCalculation');
assign('domain/calculation/periods.js','MONTHS_RU addMonths lastDayOfMonth parseOrderMonth formatDate sameDate expectedPeriods formatPeriod parsePeriodRange rangesEqual');
assign('infrastructure/excel/xml.js','NS_MAIN XML_PARSER XML_SERIALIZER parseXml elements firstElement cellKey parseCellRef columnNameToNumber columnNumberToName sheetBounds sheetCellValue refreshCellValue readCellValue findOrCreateCell clearCellChildren copyCellStyle setNumericCell setTextCell');
assign('infrastructure/excel/archive.js','loadXlsx saveXlsx forceFormulaRecalculation removeNodesByAttribute normalizeWorkbookTarget parseSharedStrings readSheetCells');
assign('infrastructure/excel/columns.js','sourceMatchers blankMatchers combinations detectColumns isMonthHeader detectUrengoyColumns detectCalculationColumns');
assign('infrastructure/excel/source.js','findSourcePeriods validateSourcePeriods isUrengoySource detectDeliveryWeeks maxMonthlySales calculateUrengoyRecommended isSourceTotalRow readSourceRows sumColumns mergeFactAndComment removeWorksheetRows monthlyValues recalculateSourceTable rebuildSourceWithChz');
assign('workflows/order/read-source.js','readSource');
assign('workflows/tyumen/merge.js','TYUMEN_COLUMNS tyumenColumns tyumenProductKey tyumenRows compatibleTyumenNames alignTyumenKeys mergeTyumenSources buildTyumenWarehousePlan');
assign('workflows/recalculation/recalculate.js','recalculateOrderTable');
assign('domain/warehouse/stock.js','warehouseTransferQuantity northTyumenFreeStock');
assign('domain/matching/source-index.js','articleKeys uniqueBySourceRow buildSourceContext sourceDuplicateGroups candidatesForArticle duplicateCandidatesForReport');
assign('domain/order/quantities.js','calculateAdjustedQuantity adjustQuantityForBrand orderForItem supplierUnitsFromPieces demandPiecesFromSupplierUnits convertOrderToSupplierUnits novacutanOrderForItem calculateMultipleAdjustedQuantity nearestMultipleValue calculateNearestMultipleAdjustedQuantity novacutanMinimumQuantity novacutanSupplierUnitSize novacutanSummaryQuantity roundSupplierPack10 novacutanSupplierOrderQuantity');
assign('domain/order/budget-rules.js','budgetOrderRules');
assign('infrastructure/excel/pricing.js','budgetSourceMetrics budgetCellFormula budgetCellPrice budgetPrices christinaLinesCache christinaProffLines setBudgetFormula budgetInvoicePriceColumn applyBudgetWorkbookPricing');
assign('workflows/budget/adapters.js','budgetReportRows');
assign('infrastructure/excel/blanks.js','genericBlankPositions duplicateArticleWarnings sourceItemsForMissingBlank detectSothysVariantBlank sothysPositions novacutanBlankPositions northBlankDetection detectSkinSynergyNorthBlank detectNameOnlyNorthBlank detectKlappBlank detectNovacutanBlank ensureNovacutanQuantityColumn prepareNovacutanQuantityCell northGenericPositions northNovacutanPositions northSplitVariantPositions workbookPriceOptions northPositions');
assign('domain/matching/novacutan.js','novacutanMatchKey chooseNovacutanCandidate novacutanPositionKey novacutanNameTokens novacutanNameScore');
assign('workflows/order/fill.js','fillWorkbook fillNovacutanWorkbook fillSplitVariantWorkbook makeUnmatchedReportRow makeReportRow');
assign('workflows/order/edits.js','normalizedBaselineQuantity parseEditValue applyEdits normalizeOrderValue applyFinalEdits');
assign('infrastructure/excel/stocks.js','addOrderBlankStocks');
assign('shared/filenames.js','outputFileName sourceOutputFileName outputExtension');
assign('domain/north/cities.js','NORTH_CITIES NORTH_TRANSFER_DISPLAY_ORDER northCityFromText');
assign('infrastructure/excel/cities.js','detectSourceCity validateNorthTyumenSourceWorkbook detectNorthCity');
assign('domain/north/allocation.js','allocateNorthNeedByCity christinaNorthSupplierQuantity defaultNorthActualSupplierOrder northTotalCityParts');
assign('domain/north/comments.js','formatNorthQuantity formatNorthCommentQuantity supplierPartsForActual northTransferParts northSupplierOrderText northPlanCommentForOrderTable novacutanNorthOrderTable northNameKey');
assign('workflows/north/availability.js','sourceKeyCandidatesForNorth readNorthTyumenAvailability northAvailabilityForTotal');
assign('workflows/north/export.js','writeNorthSummaryWorkbook transferItemsForCity transferFilesFromTotals budgetNorthWorkbookPricing');
assign('workflows/north/plan.js','addNorthTotal scopedNorthKey scopedNorthPositions northPlanRowFromTotal buildNorthPlan editedNorthTotals recalculateEditedNorthPlan actualQuantityByKey');
assign('workflows/north/order.js','finalizeNorthOrderFiles buildNorthOrderFiles');
const imports=new Map();
for(const n of ast.body.filter(n=>n.type==='ImportDeclaration'))for(const s of n.specifiers)imports.set(s.local.name,{name:s.imported?.name || '*',source:n.source.value});
const owners=new Map();const records=[];
for(const raw of ast.body.filter(n=>n.type!=='ImportDeclaration')){
  const n=raw.type==='ExportNamedDeclaration'?raw.declaration:raw;
  const names=n.type==='FunctionDeclaration'?[n.id.name]:n.declarations.map(d=>d.id.name);
  const owner=groups[names[0]];
  if(!owner||names.some(name=>groups[name]!==owner))throw Error(`Missing owner: ${names}`);
  const refs=[...bindings.values()].flatMap(v=>v.references.map(r=>({name:v.name,id:r.identifier}))).filter(r=>r.id.start>=n.start&&r.id.end<=n.end);
  const deps=[...new Set(refs.map(r=>r.name).filter(name=>!names.includes(name)))];
  const previous=ast.body[ast.body.indexOf(raw)-1];
  const prefix=source.slice(previous?.end || 0,raw.start).trim();
  const record={raw,n,names,owner,deps,prefix,text:source.slice(n.start,n.end)};records.push(record);names.forEach(name=>owners.set(name,owner));
}
const edges=new Map();for(const r of records){if(!edges.has(r.owner))edges.set(r.owner,new Set());for(const name of r.deps){const owner=owners.get(name);if(owner&&owner!==r.owner)edges.get(r.owner).add(owner);}}
const visiting=new Set(),done=new Set(),cycles=[];
function visit(owner,chain=[]){if(visiting.has(owner)){cycles.push([...chain.slice(chain.indexOf(owner)),owner]);return;}if(done.has(owner))return;visiting.add(owner);for(const dep of edges.get(owner)||[])visit(dep,[...chain,owner]);visiting.delete(owner);done.add(owner);}
for(const owner of edges.keys())visit(owner);
console.log(JSON.stringify({modules:[...edges].map(([name,deps])=>({name,deps:[...deps],lines:records.filter(r=>r.owner===name).reduce((n,r)=>n+r.text.split('\n').length,0)})),cycles},null,2));
if(process.argv.includes('--write') || process.argv.includes('--pure')){
  if(cycles.length)throw Error('Resolve import cycles before extraction');
  const selected=new Set([...edges.keys()].filter(owner=>!process.argv.includes('--pure') || owner.startsWith('domain/') || owner.startsWith('shared/')));
  for(const owner of selected){
    const items=records.filter(r=>r.owner===owner);const byPath=new Map();
    for(const name of new Set(items.flatMap(r=>r.deps))){const dep=owners.get(name);if(dep===owner)continue;const imported=imports.get(name);const p=dep?`./${path.posix.relative(path.posix.dirname(owner),dep)}`:imported?.source;if(!p)throw Error(`Unknown dependency ${name}`);if(!byPath.has(p))byPath.set(p,[]);byPath.get(p).push(imported?.name&&imported.name!==name?`${imported.name} as ${name}`:name);}
    const body=[...byPath].map(([p,names])=>`import { ${[...new Set(names)].sort().join(', ')} } from ${JSON.stringify(p)};`).join('\n')+'\n\n'+items.map(r=>`${r.prefix ? r.prefix+'\n' : ''}export ${r.text}`).join('\n\n')+'\n';
    await mkdir(`src/${path.posix.dirname(owner)}`,{recursive:true});await writeFile(`src/${owner}`,body);
  }
  const publicExports=records.filter(r=>r.raw.type==='ExportNamedDeclaration').flatMap(r=>r.names);
  if(process.argv.includes('--pure')){
    const moved=[...selected].map(owner=>`import { ${records.filter(r=>r.owner===owner).flatMap(r=>r.names).join(', ')} } from './${owner}';`).join('\n');
    const remaining=records.filter(r=>!selected.has(r.owner)).map(r=>`${r.prefix ? r.prefix+'\n' : ''}${source.slice(r.raw.start,r.raw.end)}`).join('\n\n');
    await writeFile('src/workbookProcessor.js',ast.body.filter(n=>n.type==='ImportDeclaration').map(n=>source.slice(n.start,n.end)).join('\n')+'\n'+moved+'\n'+publicExports.filter(n=>selected.has(owners.get(n))).map(n=>`export { ${n} };`).join('\n')+'\n\n'+remaining+'\n');
  }else await writeFile('src/workbookProcessor.js',publicExports.map(name=>`export { ${name} } from './${owners.get(name)}';`).join('\n')+'\n');
  await mkdir('docs/plans',{recursive:true});await writeFile('docs/plans/export-owners.json',JSON.stringify({baseline:'1e2baa1354c40156ff8f559ad5b2f2d27e119a0e',owners:Object.fromEntries(owners),publicExports,bodyHashes:Object.fromEntries(records.flatMap(r=>r.names.map(name=>[name,createHash('sha256').update(r.text).digest('hex')])))},null,2)+'\n');
}
}
