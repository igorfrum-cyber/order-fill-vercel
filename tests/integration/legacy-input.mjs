import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { parseAst } from 'rollup/parseAst';
import { read, write } from 'xlsx';
import { unzipSync } from 'fflate';
import { book } from '../fixtures/workbooks.mjs';
import { createFilesLoader } from '../../src/ui/files/loader.js';
const commit='1e2baa1354c40156ff8f559ad5b2f2d27e119a0e';
const source=execFileSync('git',['show',`${commit}:src/app.js`],{encoding:'utf8'});
const ast=parseAst(source),names=['isLegacyXls','normalizeWorkbookBytes'];
const declarations=ast.body.filter(n=>n.type==='FunctionDeclaration'&&names.includes(n.id.name));
const dir=new URL('../../test-output/',import.meta.url);await mkdir(dir,{recursive:true});
await writeFile(new URL('reference-loader.mjs',dir),"import {read as readSpreadsheet,write as writeSpreadsheet} from 'xlsx';\n"+declarations.map(n=>'export '+source.slice(n.start,n.end)).join('\n'));
const reference=await import(new URL('reference-loader.mjs',dir));
const current=createFilesLoader({elements:{},state:{},actions:{}});
const xlsx=book([['Товар','Количество','Комментарий'],['Синтетический товар',0,'Сохранить ноль'],['Вторая строка',3,'']]);
assert.equal(await current.normalizeWorkbookBytes(xlsx,'fixture.xlsx'),xlsx);
assert.equal(await current.normalizeWorkbookBytes(xlsx,'fixture.xlsm'),xlsx);
const legacy=write(read(xlsx,{type:'buffer'}),{type:'buffer',bookType:'biff8'});
const input=legacy.buffer.slice(legacy.byteOffset,legacy.byteOffset+legacy.byteLength);
const expected=await reference.normalizeWorkbookBytes(input,'fixture.xls');
const actual=await current.normalizeWorkbookBytes(input,'fixture.xls');
const parts=b=>Object.fromEntries(Object.entries(unzipSync(new Uint8Array(b))).map(([p,v])=>[p,Buffer.from(v)]));
assert.deepEqual(parts(actual),parts(expected));
assert.equal(read(actual,{type:'array'}).Sheets.Тюмень.B2.v,0);
for(const options of [{allowLegacyXls:false}]){
  let before,after;try{await reference.normalizeWorkbookBytes(input,'fixture.xls',options);}catch(e){before=e.message;}
  try{await current.normalizeWorkbookBytes(input,'fixture.xls',options);}catch(e){after=e.message;}
  assert.equal(after,before);
}
console.log('Synthetic BIFF8 XLS conversion, explicit zero, XLSX/XLSM identity and legacy rejection match original loader');
