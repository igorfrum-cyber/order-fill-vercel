import { readFile, writeFile, mkdir, rename } from 'node:fs/promises';
const moves=[['test-tyumen.mjs','integration/tyumen.mjs'],['test-budget.mjs','integration/budget.mjs'],['test-christina-lines.mjs','unit/christina-lines.mjs'],['test-workbook.mjs','private/workbook.mjs'],['check-tyumen-ui.mjs','e2e/tyumen.mjs']];
for(const [old,target] of moves){
  const text=await readFile(`scripts/${old}`,'utf8');await mkdir(`tests/${target.slice(0,target.lastIndexOf('/'))}`,{recursive:true});
  await writeFile(`tests/${target}`,text.replaceAll('../src/','../../src/'));
  await writeFile(`scripts/${old}`,`${target.startsWith('private/')?"console.log('Private workbook check: requires the documented local Downloads fixtures; missing files are a failure, not a pass.');\n":''}import '../tests/${target}';\n`);
}
