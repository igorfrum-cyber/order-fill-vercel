import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { parseAst } from 'rollup/parseAst';
async function save(path,text){await mkdir(path.slice(0,path.lastIndexOf('/')),{recursive:true});await writeFile(path,text);}
function exports(text){return parseAst(text).body.filter(n=>n.type==='ExportNamedDeclaration').flatMap(n=>n.declaration.type==='FunctionDeclaration'?[n.declaration.id.name]:n.declaration.declarations.map(d=>d.id.name));}
for(const [old,target,replacements] of [
  ['src/christinaLines.js','src/domain/budget/christina-lines.js',[]],
  ['src/budgetPlanner.js','src/domain/budget/planner.js',[["'./christinaLines.js'","'./christina-lines.js'"]]],
  ['src/budgetDialog.js','src/ui/budget/dialog.js',[["'./budgetPlanner.js'","'../../domain/budget/planner.js'"],["'./christinaLines.js'","'../../domain/budget/christina-lines.js'"]]],
  ['src/recalculateTable.js','src/ui/recalculation/controller.js',[["import {recalculateOrderTable,saveXlsx} from './workbookProcessor.js';","import {recalculateOrderTable} from '../../workflows/recalculation/recalculate.js';\nimport {saveXlsx} from '../../infrastructure/excel/archive.js';"]]],
]){
  let text=await readFile(old,'utf8');const names=exports(text);for(const [a,b] of replacements)text=text.replace(a,b);
  await save(target,text);await writeFile(old,`export { ${names.join(', ')} } from './${target.slice(4)}';\n`);
}
const pricing=await readFile('src/orderPricing.js','utf8');const ast=parseAst(pricing);
const declaration=name=>ast.body.find(n=>n.declaration?.id?.name===name || n.declaration?.declarations?.some(d=>d.id.name===name));
const create=declaration('createOrderPricing'),pure=declaration('priceOrderRows'),money=declaration('money');
const escape=ast.body.find(n=>n.type==='VariableDeclaration');
await save('src/shared/money.js',pricing.slice(money.start,money.end)+'\n');
await save('src/domain/budget/pricing.js',pricing.slice(pure.start,pure.end)+'\n');
await save('src/ui/pricing/settings.js',"import {discountValue} from '../../domain/budget/planner.js';\nimport {money} from '../../shared/money.js';\n"+pricing.slice(escape.start,escape.end)+'\n\n'+pricing.slice(pricing.indexOf('/** File selection'),create.start)+pricing.slice(create.start,create.end)+'\n');
await writeFile('src/orderPricing.js',"export {money} from './shared/money.js';\nexport {createOrderPricing} from './ui/pricing/settings.js';\nexport {priceOrderRows} from './domain/budget/pricing.js';\n");
