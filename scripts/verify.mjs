import { spawnSync } from 'node:child_process';
const commands=[['node','scripts/verify-structure.mjs'],['node','scripts/test-tyumen.mjs'],['node','scripts/test-budget.mjs'],['node','scripts/test-christina-lines.mjs'],['node','tests/integration/legacy-input.mjs'],['node','tests/parity/check.mjs'],['npm','run','build']];
for(const [cmd,...args] of commands){const r=spawnSync(cmd,args,{stdio:'inherit'});if(r.status!==0)process.exit(r.status || 1);}
console.log('Autonomous verification passed. Integration additionally requires npm run test:e2e.');
