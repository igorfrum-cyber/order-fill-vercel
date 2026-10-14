import assert from 'node:assert/strict';
import { readFile, writeFile, mkdir, access } from 'node:fs/promises';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { resolve, sep } from 'node:path';
import { unzipSync } from 'fflate';
import {functional} from './functional.mjs';
const { chromium }=await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const { PNG }=await import(process.env.PNG_MODULE || 'pngjs');
const capture=process.argv.includes('--capture');
const golden=process.env.UI_BASELINE_DIR
  ? pathToFileURL(resolve(process.env.UI_BASELINE_DIR)+sep)
  : new URL('../../test-output/ui-baseline/',import.meta.url);
const fixtures=new URL('../fixtures/input/',import.meta.url);
await mkdir(golden,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{})});
const page=await browser.newPage({viewport:{width:1440,height:1000},locale:'ru-RU',timezoneId:'Asia/Yekaterinburg'});
await page.addInitScript(()=>{const Original=Date;globalThis.Date=class extends Original{constructor(...a){super(...(a.length?a:['2026-10-13T10:00:00Z']));}static now(){return new Original('2026-10-13T10:00:00Z').valueOf();}};});
const errors=[];
const dialogs=[];
let cornerBoxes=[];
let edgeBoxes=[];
page.on('pageerror',e=>errors.push(e.message));
page.on('dialog',async d=>{dialogs.push(d.message());await d.accept();});
async function record(name,value){
  const path=new URL(name,golden);
  if(name.endsWith('.png'))await writeFile(new URL(`../../test-output/${name}`,import.meta.url),value);
  const compare=async()=>{
    const expected=await readFile(path);
    if(!name.endsWith('.png'))return assert.ok(expected.equals(Buffer.from(value)),`Baseline mismatch: ${name}`);
    const a=PNG.sync.read(expected),b=PNG.sync.read(Buffer.from(value));
    assert.equal(a.width,b.width);assert.equal(a.height,b.height);
    const pixels=new Set(),strong=new Set();let changed=0,delta=0;for(let i=0;i<a.data.length;i+=4)if(!a.data.subarray(i,i+4).equals(b.data.subarray(i,i+4))){changed++;pixels.add(i/4);let local=0;for(let c=0;c<4;c++)local=Math.max(local,Math.abs(a.data[i+c]-b.data[i+c]));delta=Math.max(delta,local);if(local>16)strong.add(i/4);}
    // Repeated baseline captures vary only at rounded raster edges in Chrome.
    assert.ok(changed<=100 && delta<=48 && strong.size<=2,`${name}: ${changed} changed pixels, maximum channel delta ${delta}, ${strong.size} strong edge pixels`);
    for(const pixel of pixels){const x=pixel%a.width+0.5,y=Math.floor(pixel/a.width)+0.5;
      const corner=cornerBoxes.some(c=>Math.abs(x-c.x)<=c.radius+2&&Math.abs(y-c.y)<=c.radius+2&&Math.abs(Math.hypot(x-c.x,y-c.y)-c.radius)<=2);
      const straight=edgeBoxes.some(b=>(x>=b.x+b.r&&x<=b.x+b.w-b.r&&Math.min(Math.abs(y-b.y),Math.abs(y-b.y-b.h))<=2)
        ||(y>=b.y+b.r&&y<=b.y+b.h-b.r&&Math.min(Math.abs(x-b.x),Math.abs(x-b.x-b.w))<=2));
      assert.ok(corner||straight,`${name}: non-border pixel at ${x},${y}`);
    }
    const components=[];const pending=new Set(pixels);while(pending.size){const queue=[pending.values().next().value];pending.delete(queue[0]);let count=0,strongCount=0;while(queue.length){const p=queue.pop();count++;if(strong.has(p))strongCount++;for(const dx of [-1,0,1])for(const dy of [-1,0,1]){const n=p+dx+dy*a.width;if(pending.delete(n))queue.push(n);}}assert.ok(count<=12,`${name}: connected difference region of ${count} pixels`);components.push({pixels:count,strong:strongCount});}
    console.log(`Pixels ${name}: count=${changed}, delta=${delta}, strong=${strong.size}, only rounded borders, components=${JSON.stringify(components)}`);
  };
  if(capture){try{await access(path);await compare();return;}catch(e){if(e.code!=='ENOENT')throw e;}await writeFile(path,value);}
  else await compare();
}
async function shot(name){
  await page.mouse.move(0,0);await page.evaluate(async()=>{document.activeElement?.blur();await document.fonts.ready;await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));});
  const snapshot=await page.evaluate(()=>[...document.body.querySelectorAll('*')].filter(el=>!['SCRIPT','STYLE'].includes(el.tagName)).map(el=>{
    const rect=el.getBoundingClientRect(),style=getComputedStyle(el),round=v=>Math.round(v*1000)/1000;
    const box=[round(rect.x+scrollX),round(rect.y+scrollY),round(rect.width),round(rect.height)];
    const radii=['borderTopLeftRadius','borderTopRightRadius','borderBottomRightRadius','borderBottomLeftRadius'].map(k=>parseFloat(style[k])||0);
    return {tag:el.tagName,id:el.id,class:el.getAttribute('class')||'',text:[...el.childNodes].filter(n=>n.nodeType===3).map(n=>n.textContent).join(''),box,radii,value:'value' in el?el.value:null,checked:'checked' in el?el.checked:null,disabled:'disabled' in el?el.disabled:null,hidden:el.hidden,download:el.getAttribute('download')};
  }));
  await record(`${name}.dom.json`,JSON.stringify(snapshot,null,2)+'\n');
  cornerBoxes=snapshot.flatMap(({box:[x,y,w,h],radii:r})=>[[x+r[0],y+r[0],r[0]],[x+w-r[1],y+r[1],r[1]],[x+w-r[2],y+h-r[2],r[2]],[x+r[3],y+h-r[3],r[3]]].filter(c=>c[2]>0&&w>0&&h>0).map(([x,y,radius])=>({x,y,radius})));
  edgeBoxes=snapshot.filter(s=>s.box[2]>0&&s.box[3]>0&&s.radii.some(r=>r>0)).map(({box:[x,y,w,h],radii})=>({x,y,w,h,r:Math.max(...radii)}));
  const native=await page.evaluate(()=>[...document.querySelectorAll('input[type=checkbox],input[type=radio]')].map(el=>{const b=el.getBoundingClientRect();return {x:b.x+scrollX,y:b.y+scrollY,w:b.width,h:b.height,r:el.type==='radio'?Math.min(b.width,b.height)/2:2};}).filter(b=>b.w&&b.h));
  edgeBoxes.push(...native);cornerBoxes.push(...native.flatMap(b=>[[b.x+b.r,b.y+b.r],[b.x+b.w-b.r,b.y+b.r],[b.x+b.w-b.r,b.y+b.h-b.r],[b.x+b.r,b.y+b.h-b.r]].map(([x,y])=>({x,y,radius:b.r}))));
  await record(`${name}.png`,await page.screenshot({fullPage:true,animations:'disabled',caret:'hide'}));console.log('UI:',name);
}
const input=(name)=>fileURLToPath(new URL(encodeURIComponent(name),fixtures));
async function download(selector,name,count){
  const expected=count || (selector==='#downloadButton'?2:1),downloads=[];
  let onDownload;
  const waiting=new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error(`Expected ${expected} downloads for ${selector}, received ${downloads.length}`)),15000);onDownload=d=>{downloads.push(d);if(downloads.length===expected){clearTimeout(timer);resolve();}};page.on('download',onDownload);});
  await page.locator(selector).click();await waiting;page.off('download',onDownload);
  for(const [index,d] of downloads.entries()){
    const bytes=await readFile(await d.path());
    const parts=Object.fromEntries(Object.entries(unzipSync(bytes)).sort(([a],[b])=>a.localeCompare(b)).map(([p,b])=>[p,Buffer.from(b).toString('base64')]));
    await record(`${name}${index?'-'+index:''}.json`,JSON.stringify({filename:d.suggestedFilename(),parts},null,2)+'\n');
  }
}
try {
  await page.goto(process.env.TEST_URL || 'http://127.0.0.1:3210/');
  if(capture){const original=await (await page.request.get(new URL('/src/app.js',page.url()).href)).text();assert.ok(original.includes('function installBudgetControls(')&&original.includes('let currentResults ='), 'Capture must use the original monolithic baseline server, never the refactored application');}
  await record('environment.json',JSON.stringify({browser:browser.version(),viewport:page.viewportSize(),...await page.evaluate(()=>({scale:devicePixelRatio,font:getComputedStyle(document.body).fontFamily,language:navigator.language,timezone:Intl.DateTimeFormat().resolvedOptions().timeZone}))},null,2)+'\n');
  for(const [mode,selector] of [['order','#orderModeButton'],['north','#northModeButton'],['warehouse','#warehouseModeButton'],['recalculation','#recalculateModeButton']]){
    await page.locator(selector).click();await page.setViewportSize({width:1440,height:1000});await shot(`${mode}-desktop`);
    await page.setViewportSize({width:390,height:844});await shot(`${mode}-mobile`);
  }
  await page.setViewportSize({width:1440,height:1000});await page.locator('#orderModeButton').click();
  await page.locator('#brandSelect').selectOption('skin_synergy');await page.locator('#orderMonth').fill('2026-10');
  await page.locator('#sourceFile').setInputFiles(input('Тюмень.xlsx'));await page.locator('#blankFile').setInputFiles(input('Бланк.xlsx'));
  await page.locator('#uploadForm .order-pricing[data-ready=true]').waitFor();await page.locator('#uploadForm [data-column]').selectOption('4');
  await page.locator('#uploadForm button[type=submit]').click();await page.locator('#result:not(.hidden)').waitFor();
  await shot('order-filled');
  const quantity=page.locator('#reportBody .qty-input').first();
  await quantity.fill('11');await page.locator('#reportBody .comment-input').first().fill('Ручной комментарий');
  await page.locator('#result .budget-panel button').first().click();await page.locator('.budget-dialog').waitFor();
  await page.locator('.budget-dialog [data-target]').fill('1500');await page.locator('.budget-dialog [data-run]').click();await shot('budget-preview');
  await page.locator('.budget-dialog [data-apply]').click();await page.locator('.budget-dialog').waitFor({state:'detached'});
  await page.locator('#result .budget-panel button').nth(1).click();assert.equal(await quantity.inputValue(),'11');
  await download('#downloadButton','order-download');
  await page.locator('#warehouseModeButton').click();await page.locator('#warehouseBrand').selectOption('skin_synergy');
  await page.locator('#officeTransferFile').setInputFiles(input('Тюмень.xlsx'));await page.locator('#warehouseTransferFile').setInputFiles(input('Склад.xlsx'));
  await page.locator('#warehouseForm button').click();await page.locator('#warehouseResult:not(.hidden)').waitFor();await shot('warehouse-filled');
  await download('#warehouseDownload','warehouse-download');
  await page.locator('#northModeButton').click();await page.locator('#northBrandSelect').selectOption('skin_synergy');
  await page.locator('#northFileInput').setInputFiles(input('Сургут.xlsx'));await page.locator('#northSourceFile').setInputFiles(input('Тюмень.xlsx'));
  await page.locator('#northForm .order-pricing[data-ready=true]').waitFor();await page.locator('#northForm [data-column]').selectOption('4');
  await page.locator('#northForm button[type=submit]').click();await page.locator('#northMergeContinue').click();await page.locator('#northResult:not(.hidden)').waitFor();await shot('north-filled');
  await page.locator('#recalculateModeButton').click();await page.locator('#recalculateSection [data-brand]').selectOption('skin_synergy');
  await page.locator('#recalculateSection [data-month]').fill('2026-10');await page.locator('#recalculateSection [data-office]').setInputFiles(input('Тюмень.xlsx'));
  await page.locator('#recalculateSection button[type=submit]').click();await page.locator('#recalculateSection [data-result]').waitFor({state:'visible'});await shot('recalculation-filled');
  await download('#recalculateSection [data-download]','recalculation-download');
  await record('dialogs.json',JSON.stringify(dialogs,null,2)+'\n');
  await functional({page,input,record,download,dialogs});
  assert.deepEqual(errors,[]);
  console.log(capture?'Captured baseline UI and downloads':'Desktop/mobile UI, budget apply/undo, scenarios and downloads match baseline');
} catch(e) { console.error('Browser failure:',errors,await page.locator('#status').textContent()); throw e; }
finally {await browser.close();}
