import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { parseAst } from 'rollup/parseAst';
import { analyze } from 'eslint-scope';
async function files(dir){return (await Promise.all((await readdir(dir,{withFileTypes:true})).map(e=>e.isDirectory()?files(`${dir}/${e.name}`):`${dir}/${e.name}`))).flat();}
const js=(await files('src')).filter(p=>p.endsWith('.js'));
const facades=new Set(['src/workbookProcessor.js','src/budgetPlanner.js','src/budgetDialog.js','src/orderPricing.js','src/christinaLines.js','src/recalculateTable.js']);
const graph=new Map(),declared=new Map();
for(const file of js){
  const text=await readFile(file,'utf8'),ast=parseAst(text);
  const deps=[];
  for(const node of ast.body){
    if(node.type==='ExportNamedDeclaration'&&node.declaration){const d=node.declaration;for(const name of d.id?[d.id.name]:d.declarations.map(v=>v.id.name))declared.set(`${file}:${name}`,text.slice(d.start,d.end));}
    if(!node.source)continue;
    const value=node.source.value;
    if(!value.startsWith('.'))continue;
    const target=path.normalize(path.join(path.dirname(file),value));await readFile(target);
    if(target.endsWith('.js'))deps.push(target);
    if(!facades.has(file))assert.ok(!facades.has(target),`Internal module ${file} imports facade ${target}`);
    if(file.startsWith('src/domain/'))assert.ok(target.startsWith('src/domain/')||target.startsWith('src/shared/'),`Domain dependency: ${file} -> ${target}`);
  }
  if(file.startsWith('src/domain/')){
    function walk(n){if(!n||typeof n!=='object')return;if(n.type)n.range=[n.start,n.end];for(const [k,v] of Object.entries(n))if(k!=='range'){if(Array.isArray(v))v.forEach(walk);else if(v?.type)walk(v);}}walk(ast);
    const scopes=analyze(ast,{ecmaVersion:2024,sourceType:'module'});
    const globals=scopes.globalScope.through.map(r=>r.identifier.name);
    for(const name of ['window','document','File','Blob','DOMParser','XMLSerializer','fetch','XMLHttpRequest','URL'])assert.ok(!globals.includes(name),`${file} refers to forbidden domain global ${name}`);
  }
  graph.set(file,deps);
}
const visiting=new Set(),visited=new Set();
function visit(file){assert.ok(!visiting.has(file),`Import cycle at ${file}`);if(visited.has(file))return;visiting.add(file);for(const dep of graph.get(file)||[])visit(dep);visiting.delete(file);visited.add(file);}
for(const file of js)visit(file);
const registry=JSON.parse(await readFile('docs/plans/export-owners.json','utf8'));
for(const [name,hash] of Object.entries(registry.bodyHashes)){
  const text=declared.get(`src/${registry.owners[name]}:${name}`);assert.ok(text,`Missing owner ${name}`);
  assert.equal(createHash('sha256').update(text).digest('hex'),hash,`Moved body changed: ${name}`);
}
const protectedFiles=JSON.parse(await readFile('tests/parity/protected-files.json','utf8'));
for(const [file,hash] of Object.entries(protectedFiles))assert.equal(createHash('sha256').update(await readFile(file)).digest('hex'),hash,`Protected UI/config file changed: ${file}`);
const docs=(await files('docs')).filter(p=>p.endsWith('.md'));
const index=await readFile('docs/README.md','utf8');
for(const file of [...docs,'README.md']){
  const text=await readFile(file,'utf8');
  if(file!=='README.md')assert.ok(/Статус:/.test(text),`Missing status: ${file}`);
  if(file.startsWith('docs/')&&file!=='docs/README.md'&&!file.startsWith('docs/archive/')&&!/Статус: заменён/.test(text))assert.ok(index.includes(file.slice(5)),`Document not indexed: ${file}`);
  for(const match of text.matchAll(/\[[^\]]*\]\(([^)]+)\)/g)){
    const link=match[1].replace(/^<|>$/g,'').split('#')[0];if(!link||/^[a-z][a-z0-9+.-]*:/i.test(link))continue;
    await readFile(path.resolve(path.dirname(file),decodeURIComponent(link)));
  }
}
console.log(`Structure: ${js.length} modules, no import cycles, pure domain, exact moved bodies, unchanged UI/config, documentation links/index: passed`);
