import {execFileSync} from 'node:child_process';
import {mkdir,writeFile,readFile} from 'node:fs/promises';
const commit=JSON.parse(await readFile('tests/parity/baseline.json','utf8')).commit;
const files=['index.html','package.json',...execFileSync('git',['ls-tree','--name-only',`${commit}:src`],{encoding:'utf8'}).trim().split('\n').map(name=>`src/${name}`)];
for(const file of files){await mkdir(`test-output/ui-reference/${file.split('/').slice(0,-1).join('/')}`,{recursive:true});await writeFile(`test-output/ui-reference/${file}`,execFileSync('git',['show',`${commit}:${file}`]));}
console.log('Prepared unchanged UI source from',commit);
