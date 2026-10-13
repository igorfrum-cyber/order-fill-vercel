import { spawnSync } from 'node:child_process';
import { writeFile, mkdir } from 'node:fs/promises';
const baseline=process.env.BASELINE_URL || 'http://127.0.0.1:3213/';
const current=process.env.TEST_URL || 'http://127.0.0.1:3212/';
const results=[],dir=`test-output/ui-review/${process.env.UI_RUN_ID || Date.now()}`;await mkdir(dir,{recursive:true});
for(const [label,url] of [['baseline',baseline],['refactor',current]])for(let run=1;run<=3;run++){
  const result=spawnSync(process.execPath,['tests/e2e/parity.mjs'],{env:{...process.env,TEST_URL:url},encoding:'utf8',timeout:180000});
  const log=(result.stdout||'')+(result.stderr||'');await writeFile(`${dir}/${label}-${run}.log`,log);
  const pixels=[...log.matchAll(/Pixels (.*): count=(\d+), delta=(\d+), strong=(\d+)/g)].map(([,name,count,delta,strong])=>({name,count:Number(count),delta:Number(delta),strong:Number(strong)}));
  results.push({label,run,url,exitCode:result.status,pixels});await writeFile(`${dir}/results.json`,JSON.stringify(results,null,2)+'\n');
  console.log(`${label} run ${run}: exit=${result.status}, max pixels=${Math.max(...pixels.map(p=>p.count),0)}, max channel delta=${Math.max(...pixels.map(p=>p.delta),0)}`);
  if(result.status!==0){console.error(log);process.exit(1);}
}
console.log('Three baseline-self and three refactor-baseline runs passed; exact DOM, bounded edge-only raster changes, functional states and downloads.');
