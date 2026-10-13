import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

export async function functional({page,input,record,download,dialogs}) {
  const initialDialogs=dialogs.length;
  await page.reload();
  await page.locator('#brandSelect').selectOption('skin_synergy');await page.locator('#orderMonth').fill('2026-10');
  await page.locator('#sourceFile').setInputFiles(input('Тюмень.xlsx'));await page.locator('#blankFile').setInputFiles(input('Бланк.xlsx'));
  await page.locator('#uploadForm .order-pricing[data-ready=true]').waitFor();await page.locator('#uploadForm [data-column]').selectOption('4');
  await page.locator('#uploadForm button[type=submit]').click();await page.locator('#result:not(.hidden)').waitFor();
  await page.locator('#reportBody .qty-input').first().fill('11');
  await page.locator('#reportBody .comment-input').first().fill('');
  const dialogStart=dialogs.length,downloads=[];const listener=d=>downloads.push(d.suggestedFilename());page.on('download',listener);
  await page.locator('#downloadButton').click();
  await record('functional-comment-block.json',JSON.stringify({dialogs:dialogs.slice(dialogStart),downloads,disabled:await page.locator('#downloadButton').isDisabled(),commentRequired:await page.locator('#reportBody .comment-input').first().getAttribute('required')},null,2)+'\n');
  assert.equal(downloads.length,0);assert.ok(dialogs.length>dialogStart);page.off('download',listener);

  await page.locator('#northModeButton').click();await page.locator('#northBrandSelect').selectOption('skin_synergy');
  await page.locator('#northFileInput').setInputFiles(input('Сургут.xlsx'));await page.locator('#northSourceFile').setInputFiles(input('Тюмень.xlsx'));
  await page.locator('#northForm .order-pricing[data-ready=true]').waitFor();await page.locator('#northForm [data-column]').selectOption('4');
  await page.locator('#northForm button[type=submit]').click();await page.locator('#northMergeContinue').click();await page.locator('#northResult:not(.hidden)').waitFor();
  await page.locator('#northPlanBody .north-city-input').first().fill('12');
  await page.locator('#northPlanBody .north-actual-input').first().fill('30');
  await download('#northDownloadButton','functional-north',2);
  await record('functional-north-state.json',JSON.stringify({city:await page.locator('#northPlanBody .north-city-input').first().inputValue(),actual:await page.locator('#northPlanBody .north-actual-input').first().inputValue(),comment:await page.locator('#northPlanBody').innerText(),status:await page.locator('#northStatus').innerText()},null,2)+'\n');

  for(const change of ['brand','file']){
    await page.reload();await page.locator('#warehouseModeButton').click();await page.locator('#warehouseBrand').selectOption('skin_synergy');
    await page.locator('#officeTransferFile').setInputFiles(input('Тюмень.xlsx'));await page.locator('#warehouseTransferFile').setInputFiles(input('Склад.xlsx'));
    await page.evaluate(()=>{
      const read=File.prototype.arrayBuffer;
      window.__heldRead=false;
      File.prototype.arrayBuffer=function(){
        if(this.name==='Тюмень.xlsx'&&!window.__heldRead){window.__heldRead=true;return new Promise(resolve=>{window.__releaseRead=()=>{File.prototype.arrayBuffer=read;return read.call(this).then(resolve);};});}
        return read.call(this);
      };
    });
    await page.locator('#warehouseForm button').click();await page.waitForFunction(()=>window.__heldRead);
    if(change==='brand')await page.locator('#warehouseBrand').selectOption('angiopharm');
    else await page.locator('#officeTransferFile').setInputFiles({name:'Тюмень новая.xlsx',mimeType:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',buffer:await readFile(input('Тюмень.xlsx'))});
    await page.evaluate(()=>window.__releaseRead());
    await page.waitForFunction(()=>!document.querySelector('#warehouseForm button').disabled);
    const state={visible:await page.locator('#warehouseResult').isVisible(),rows:await page.locator('#warehouseRows tr').count(),brand:await page.locator('#warehouseBrand').inputValue(),file:await page.locator('#officeTransferName').innerText()};
    await record(`functional-stale-${change}.json`,JSON.stringify(state,null,2)+'\n');assert.equal(state.visible,false);assert.equal(state.rows,0);
  }
  await record('functional-dialogs.json',JSON.stringify(dialogs.slice(initialDialogs),null,2)+'\n');
  console.log('Missing-comment download block, edited North exports, stale warehouse file/brand results: passed');
}
