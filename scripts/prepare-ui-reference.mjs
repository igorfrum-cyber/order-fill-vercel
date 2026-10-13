import {execFileSync} from 'node:child_process';
import {mkdir,writeFile} from 'node:fs/promises';
const commit='1e2baa1354c40156ff8f559ad5b2f2d27e119a0e';
const files=['index.html','package.json',...execFileSync('git',['ls-tree','--name-only',`${commit}:src`],{encoding:'utf8'}).trim().split('\n').map(name=>`src/${name}`)];
for(const file of files){await mkdir(`test-output/ui-reference/${file.split('/').slice(0,-1).join('/')}`,{recursive:true});await writeFile(`test-output/ui-reference/${file}`,execFileSync('git',['show',`${commit}:${file}`]));}
console.log('Prepared unchanged UI source from',commit);
