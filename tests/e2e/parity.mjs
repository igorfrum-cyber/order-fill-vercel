import assert from 'node:assert/strict';
import { readFile, writeFile, mkdir, access } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { unzipSync } from 'fflate';
const { chromium }=await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const { PNG }=await import(process.env.PNG_MODULE || 'pngjs');
const capture=process.argv.includes('--capture');
const golden=new URL('../parity/ui/',import.meta.url);
const fixtures=new URL('../fixtures/input/',import.meta.url);
await mkdir(golden,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{})});
const page=await browser.newPage({viewport:{width:1440,height:1000},locale:'ru-RU',timezoneId:'Asia/Yekaterinburg'});
await page.addInitScript(()=>{const Original=Date;globalThis.Date=class extends Original{constructor(...a){super(...(a.length?a:['2026-10-13T10:00:00Z']));}static now(){return new Original('2026-10-13T10:00:00Z').valueOf();}};});
const errors=[];
const dialogs=[];
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
    let changed=0,delta=0;for(let i=0;i<a.data.length;i+=4)if(!a.data.subarray(i,i+4).equals(b.data.subarray(i,i+4))){changed++;for(let c=0;c<4;c++)delta=Math.max(delta,Math.abs(a.data[i+c]-b.data[i+c]));}
    // Repeated baseline captures vary only at rounded raster edges in Chrome.
    assert.ok(changed<=50 && delta<=2,`${name}: ${changed} changed pixels, maximum channel delta ${delta} (limits 50/2, no masks)`);
  };
  if(capture){try{await access(path);await compare();return;}catch(e){if(e.code!=='ENOENT')throw e;}await writeFile(path,value);}
  else await compare();
}
async function shot(name){await page.mouse.move(0,0);await page.evaluate(async()=>{document.activeElement?.blur();await document.fonts.ready;await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));});await record(`${name}.png`,await page.screenshot({fullPage:true,animations:'disabled',caret:'hide'}));console.log('UI:',name);}
const input=(name)=>fileURLToPath(new URL(encodeURIComponent(name),fixtures));
async function download(selector,name){
  const expected=selector==='#downloadButton'?2:1,downloads=[];
  let onDownload;
  const waiting=new Promise(resolve=>{onDownload=d=>{downloads.push(d);if(downloads.length===expected)resolve();};page.on('download',onDownload);});
  await page.locator(selector).click();await waiting;page.off('download',onDownload);
  for(const [index,d] of downloads.entries()){
    const bytes=await readFile(await d.path());
    const parts=Object.fromEntries(Object.entries(unzipSync(bytes)).sort(([a],[b])=>a.localeCompare(b)).map(([p,b])=>[p,Buffer.from(b).toString('base64')]));
    await record(`${name}${index?'-'+index:''}.json`,JSON.stringify({filename:d.suggestedFilename(),parts},null,2)+'\n');
  }
}
try {
  await page.goto(process.env.TEST_URL || 'http://127.0.0.1:3210/');
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
  assert.deepEqual(errors,[]);
  console.log(capture?'Captured baseline UI and downloads':'Desktop/mobile UI, budget apply/undo, scenarios and downloads match baseline');
} catch(e) { console.error('Browser failure:',errors,await page.locator('#status').textContent()); throw e; }
finally {await browser.close();}
