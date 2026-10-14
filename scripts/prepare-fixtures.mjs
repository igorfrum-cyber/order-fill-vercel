import { mkdir, writeFile } from 'node:fs/promises';
import { source, blank, macroBook } from '../tests/fixtures/workbooks.mjs';
const dir=new URL('../tests/fixtures/input/',import.meta.url);
await mkdir(dir,{recursive:true});
for(const [name,bytes] of Object.entries({
  'Тюмень.xlsx':source(), 'Склад.xlsx':source([{article:'P1',name:'Крем 50 мл',stock:90,sales:0}]),
  'Бланк.xlsx':blank(), 'Сургут.xlsx':blank('skin_synergy','Сургут',10),
  'Macros.xlsm':macroBook(blank()),
})) await writeFile(new URL(encodeURIComponent(name),dir),bytes);
