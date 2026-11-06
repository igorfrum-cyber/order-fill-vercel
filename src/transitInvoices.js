import { detectColumns, sheetBounds, sheetCellValue, isSourceTotalRow, normalizeArticle, normalizeHeader, parseNumber, similarity, levissimeVolumeSignature, saveXlsx, loadXlsx, setNumericCell } from './workbookProcessor.js';

function validateInvoice(rows, label) {
  if (!rows.length) throw Error(`В счёте «${label}» не найдены позиции. Сканированные PDF пока не поддерживаются.`);
  for (const row of rows) if (!row.article || !row.name || !Number.isFinite(row.quantity) || row.quantity < 0) throw Error(`Не удалось прочитать позицию счёта «${label}».`);
  return rows;
}
function invoiceIdentity(text){
  const match=text.match(/сч[её]т(?:\s+на\s+оплату)?\s*№\s*([^\s]+)\s+от\s+(\d{2}\.\d{2}\.\d{4})/iu);
  return match?`${match[1]}:${match[2]}`:null;
}
export function parseInvoiceExcel(workbook, label) {
  const rows=[];let brandText=label;
  for(const sheet of workbook.sheets){
    const bounds=sheetBounds(sheet);let columns,headerRow;
    for(let row=1;row<=Math.min(bounds.maxRow,100);row++){
      const found={};
      for(let col=1;col<=bounds.maxColumn;col++){
        const raw=String(sheetCellValue(sheet,row,col)||'');brandText+=' '+raw;
        const h=normalizeHeader(raw);
        if(/артикул|^арт$/u.test(h))found.article=col;
        if(/наименование|номенклатура|название|^товар$/u.test(h))found.name=col;
        if(/^(кол во|кол-во|количество|заказано|заказ)$/u.test(h))found.quantity=col;
      }
      if(found.article&&found.name&&found.quantity){columns=found;headerRow=row;break;}
    }
    if(!columns)continue;
    for(let row=headerRow+1;row<=bounds.maxRow;row++){
      const article=String(sheetCellValue(sheet,row,columns.article)||'').trim();
      const name=String(sheetCellValue(sheet,row,columns.name)||'').trim();
      if(!article||/^(итого|всего)(\s|$)/iu.test(name))continue;
      const quantity=parseNumber(sheetCellValue(sheet,row,columns.quantity));
      if(quantity===null)throw Error(`Не прочитано количество: ${article}, файл «${label}».`);
      rows.push({article,name,quantity});
    }
  }
  if(!/ангиофарм|angiopharm/iu.test(brandText))throw Error(`В счёте «${label}» не подтверждён ANGIOPHARM.`);
  rows.invoiceId=invoiceIdentity(brandText);
  return validateInvoice(rows,label);
}

export async function parseInvoicePdf(bytes,label){
  const pdfjs=await import('pdfjs-dist/build/pdf.mjs');
  pdfjs.GlobalWorkerOptions.workerSrc=(await import('pdfjs-dist/build/pdf.worker.min.mjs?url')).default;
  const pdf=await pdfjs.getDocument({data:new Uint8Array(bytes),isEvalSupported:false}).promise;
  const rows=[];let articleX,quantityX,nameX,unitX,documentText='';
  try{
    for(let pageNumber=1;pageNumber<=pdf.numPages;pageNumber++){
      const page=await pdf.getPage(pageNumber),content=await page.getTextContent();
      const items=content.items.filter(x=>x.str?.trim()).map(x=>({text:x.str.trim(),x:x.transform[4],y:x.transform[5]}));
      documentText+=' '+items.map(x=>x.text).join(' ');
      const header=items.find(x=>/^арт\.?$/iu.test(x.text));
      if(header){
        articleX=header.x;
        const h=items.filter(x=>Math.abs(x.y-header.y)<4);
        quantityX=h.find(x=>/кол.?во|количество/iu.test(x.text))?.x;
        nameX=articleX+32;
        unitX=h.find(x=>/единица/iu.test(x.text))?.x;
      }
      if([articleX,quantityX,nameX,unitX].some(x=>x===undefined))continue;
      const anchors=items.filter(x=>x.x<articleX-5&&/^\d+$/.test(x.text)&&items.some(a=>a.x>=articleX-8&&a.x<nameX-5&&Math.abs(a.y-x.y)<4));
      for(let index=0;index<anchors.length;index++){
        const anchor=anchors[index];
        const same=items.filter(x=>Math.abs(x.y-anchor.y)<4);
        const article=same.find(x=>x.x>=articleX-8&&x.x<nameX-5)?.text;
        const quantityItem=same.find(x=>x.x>unitX+25&&/^\d+(?:[.,]\d+)?$/.test(x.text));
        if(!article||!quantityItem)throw Error(`Не удалось разобрать строку ${anchor.text} PDF «${label}».`);
        const above=index===0?anchor.y+18:(anchors[index-1].y+anchor.y)/2;
        const below=index===anchors.length-1?anchor.y-18:(anchor.y+anchors[index+1].y)/2;
        const name=items.filter(x=>x.x>=nameX-8&&x.x<unitX-8&&x.y<=above&&x.y>below).sort((a,b)=>b.y-a.y||a.x-b.x).map(x=>x.text).join(' ');
        rows.push({article,name,quantity:parseNumber(quantityItem.text)});
      }
    }
  }finally{await pdf.destroy();}
  if(!/ангиофарм|angiopharm/iu.test(documentText))throw Error(`В PDF «${label}» не подтверждён ANGIOPHARM.`);
  const count=documentText.match(/Всего наименований\s+(\d+)/iu);
  if(count&&Number(count[1])!==rows.length)throw Error(`В PDF «${label}» прочитано ${rows.length} позиций из ${count[1]}. Проверьте формат счёта.`);
  rows.invoiceId=invoiceIdentity(documentText);
  return validateInvoice(rows,label);
}

const cleanName=name=>normalizeHeader(name).replace(/^чз\s*\+?\s*/u,'').replace(/^ан\s+/u,'').replace(/[.,]/g,' ').replace(/\s+/g,' ').trim();
const nameTokens=name=>cleanName(name).replace(/\d+(?:[.,]\d+)?\s*(мл|гр|г|л)(?=\s|$)/gu,' ').split(' ').filter(x=>x&&!['angiopharm','на','голову','и'].includes(x)).sort().join(' ');
export function buildTransitPlan(workbook,invoices){
  const detection=detectColumns(workbook,'source'),{sheet,columns}=detection,bounds=sheetBounds(sheet);
  const products=[];
  for(let row=detection.headerRow+1;row<=bounds.maxRow;row++){
    if(isSourceTotalRow(detection,row,bounds.maxColumn))continue;
    const name=String(sheetCellValue(sheet,row,columns.name)||'').trim();if(!name)continue;
    products.push({row,name,article:String(sheetCellValue(sheet,row,columns.article)||'').trim()});
  }
  const totals=new Map();
  for(const invoice of invoices)for(const item of invoice.rows){
    const key=normalizeArticle(item.article,{preserveHyphen:true});
    if(!totals.has(key))totals.set(key,{...item,quantity:0,files:[]});
    const total=totals.get(key);if(cleanName(total.name)!==cleanName(item.name))total.conflicting=true;total.quantity+=item.quantity;total.files.push(invoice.name);
  }
  const used=new Set(),plan=[];
  for(const item of totals.values()){
    const key=normalizeArticle(item.article,{preserveHyphen:true});
    let candidates=products.filter(p=>normalizeArticle(p.article,{preserveHyphen:true})===key);
    candidates=candidates.filter(p=>!levissimeVolumeSignature(p.name)||!levissimeVolumeSignature(item.name)||levissimeVolumeSignature(p.name)===levissimeVolumeSignature(item.name));
    candidates=candidates.filter(p=>similarity(cleanName(p.name),cleanName(item.name))>=0.4 || (key==='SM60'&&cleanName(p.name)==='пробник'));
    if(!candidates.length)candidates=products.filter(p=>!p.article
      && (!levissimeVolumeSignature(p.name)||!levissimeVolumeSignature(item.name)||levissimeVolumeSignature(p.name)===levissimeVolumeSignature(item.name))
      && (cleanName(p.name)===cleanName(item.name)||nameTokens(p.name)===nameTokens(item.name)));
    const chz=candidates.filter(p=>/^чз\s/iu.test(p.name));
    if(chz.length)candidates=chz;
    if(candidates.length>1){const exact=candidates.filter(p=>cleanName(p.name)===cleanName(item.name));if(exact.length===1)candidates=exact;}
    const target=!item.conflicting&&candidates.length===1&&!used.has(candidates[0].row)?candidates[0]:null;
    if(target)used.add(target.row);
    plan.push({...item,targetRow:target?.row??null,targetName:target?.name||'',warning:target?'':'Не удалось однозначно сопоставить. Рассчитайте вручную.'});
  }
  return {detection,products,rows:plan};
}
export function applyTransitPlan(workbook,plan){
  const output=loadXlsx(saveXlsx(workbook)),detection=detectColumns(output,'source');
  for(const row of plan.rows)if(!Number.isFinite(row.quantity)||row.quantity<0)throw Error('Проверьте количества всех позиций счетов, включая несопоставленные.');
  // Uploaded invoices replace the entire previous transit snapshot.
  for(const product of plan.products)setNumericCell(detection.sheet,product.row,detection.columns.inTransit,0);
  let total=0;const assigned=new Set();
  for(const row of plan.rows)if(row.targetRow!==null){
    if(assigned.has(row.targetRow))throw Error('Несколько позиций назначены одной строке таблицы. Уточните соответствия.');
    assigned.add(row.targetRow);
    if(!Number.isFinite(row.quantity)||row.quantity<0)throw Error('Количество в пути должно быть неотрицательным числом.');
    setNumericCell(detection.sheet,row.targetRow,detection.columns.inTransit,row.quantity);total+=row.quantity;
  }
  const bounds=sheetBounds(detection.sheet);
  for(let row=detection.headerRow+1;row<=bounds.maxRow;row++)if(isSourceTotalRow(detection,row,bounds.maxColumn))setNumericCell(detection.sheet,row,detection.columns.inTransit,total);
  return output;
}
