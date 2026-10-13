import { writeFile, mkdir, access } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import * as api from '../src/workbookProcessor.js';
import { scenarios } from '../tests/parity/scenarios.mjs';
const target=new URL('../tests/parity/baseline.json',import.meta.url);
try { await access(target); throw Error('Baseline already exists; refusing to overwrite it.'); } catch(e) { if(e.code!=='ENOENT')throw e; }
const commit=execFileSync('git',['rev-parse','HEAD'],{encoding:'utf8'}).trim();
await mkdir(new URL('../tests/parity/',import.meta.url),{recursive:true});
await writeFile(target,JSON.stringify({commit,normalization:'ZIP container metadata only; every uncompressed part is compared byte-for-byte.',data:await scenarios(api)},null,2)+'\n');
console.log('Captured immutable baseline',commit);
