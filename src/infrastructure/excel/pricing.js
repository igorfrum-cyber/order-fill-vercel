import { NS_MAIN, XML_PARSER, cellKey, columnNameToNumber, columnNumberToName, elements, findOrCreateCell, firstElement, parseCellRef, setNumericCell, setTextCell, sheetBounds, sheetCellValue } from "./xml.js";
import { detectDeliveryWeeks, monthlyValues } from "./source.js";
import { detectCalculationColumns } from "./columns.js";
import { calculateTargetNew, newProductCalculation, normalizeCategory } from "./../../domain/calculation/formulas.js";
import { asText, parseNumber } from "./../../shared/text.js";
import { strFromU8 } from "fflate";
import { loadXlsx, saveXlsx } from "./archive.js";

export function budgetSourceMetrics(detection, row) {
  const c = detectCalculationColumns(detection, { required: false });
  if (!c) return { demand: 0, category: '', delivery: 0.25 };
  const values = monthlyValues(detection.sheet, row, c.salesColumns);
  const weeks = Math.max(1, detectDeliveryWeeks({ sheets: [detection.sheet] }) || 1);
  const novelty = newProductCalculation(values, weeks);
  return {
    demand: novelty?.monthlyNeed ?? calculateTargetNew(values),
    category: normalizeCategory(sheetCellValue(detection.sheet, row, c.category)).split('/')[0],
    delivery: weeks * 0.25,
  };
}

// Only arithmetic and same-sheet references are accepted; workbook text is never executed.
export function budgetCellFormula(sheet, cell) {
  if (!cell) return '';
  const formulaNode = firstElement(cell.node, 'f');
  let formula = formulaNode?.textContent;
  if (!formula && formulaNode?.getAttribute('t') === 'shared') {
    const master = [...sheet.cells.values()].find(c => {
      const f = firstElement(c.node, 'f');
      return f?.getAttribute('si') === formulaNode.getAttribute('si') && f.textContent;
    });
    if (master) formula = firstElement(master.node, 'f').textContent.replace(/(\$?)([A-Z]+)(\$?)(\d+)/g, (_, ca, col, ra, row) => {
      const c = columnNameToNumber(col) + (ca ? 0 : cell.col-master.col);
      const r = Number(row) + (ra ? 0 : cell.row-master.row);
      return `${ca}${columnNumberToName(c)}${ra}${r}`;
    });
  }
  return formula || '';
}

export function budgetCellPrice(sheet, ref, seen = new Set()) {
  if (seen.has(ref)) return null;
  const cell = sheet.cells.get(ref);
  if (!cell) return null;
  const formula = budgetCellFormula(sheet, cell);
  if (!formula) return parseNumber(cell.value);
  seen = new Set([...seen, ref]);
  const aggregate = /^(SUM|SUMPRODUCT)\(([^()]*)\)$/i.exec(formula);
  if (aggregate) {
    const ranges = aggregate[2].split(',').map(part => {
      const refs = part.trim().replace(/\$/g, '').split(':');
      if (!refs.every(r => /^[A-Z]+\d+$/.test(r))) return null;
      const a = parseCellRef(refs[0]), b = parseCellRef(refs[1] || refs[0]);
      if (b.row < a.row || b.col < a.col || (b.row-a.row+1)*(b.col-a.col+1) > 100000) return null;
      const values = [];
      for (let r=a.row; r<=b.row; r++) for (let c=a.col; c<=b.col; c++) values.push(budgetCellPrice(sheet,cellKey(r,c),seen) ?? 0);
      return values;
    });
    if (ranges.every(Boolean)) {
      if (aggregate[1].toUpperCase() === 'SUM') return ranges.flat().reduce((s,v)=>s+v,0);
      if (ranges.every(r=>r.length===ranges[0].length)) return ranges[0].reduce((s,_,i)=>s+ranges.reduce((p,r)=>p*r[i],1),0);
    }
    return parseNumber(cell.value);
  }
  const round = /^ROUND\((.*),\s*(\d+)\)$/i.exec(formula);
  const sourceExpression = round ? round[1] : formula;
  let invalid = false;
  const expression = sourceExpression.replace(/\$?([A-Z]+)\$?(\d+)/g, (_, col, row) => {
    const address = parseCellRef(`${col}${row}`);
    const key = cellKey(address.row, address.col);
    const referenced = sheet.cells.get(key);
    const value = !referenced || (referenced.value === '' && !budgetCellFormula(sheet, referenced)) ? 0 : budgetCellPrice(sheet, key, seen);
    if (value == null) invalid = true;
    return `(${value ?? 0})`;
  }).replace(/%/g, '/100');
  if (invalid || /[^\d.\s()+*/-]/.test(expression)) return parseNumber(cell.value);
  const tokens = expression.match(/\d+(?:\.\d*)?|\.\d+|[()+*/-]/g) || [];
  let i = 0;
  function atom() {
    if (tokens[i] === '-') { i++; return -atom(); }
    if (tokens[i] === '+') { i++; return atom(); }
    if (tokens[i] === '(') { i++; const v = sum(); if (tokens[i++] !== ')') throw Error(); return v; }
    const v = Number(tokens[i++]); if (!Number.isFinite(v)) throw Error(); return v;
  }
  function product() { let v = atom(); while (tokens[i] === '*' || tokens[i] === '/') { const op = tokens[i++], b = atom(); v = op === '*' ? v*b : v/b; } return v; }
  function sum() { let v = product(); while (tokens[i] === '+' || tokens[i] === '-') { const op = tokens[i++], b = product(); v = op === '+' ? v+b : v-b; } return v; }
  try {
    const v = sum();
    if (i !== tokens.length || !Number.isFinite(v)) return null;
    return round ? Math.round((v + Number.EPSILON) * 10 ** Number(round[2])) / 10 ** Number(round[2]) : v;
  } catch { return null; }
}

export function budgetPrices(detection, row, quantityCol) {
  const result = [];
  if (detection.blocks) {
    const block = detection.blocks.find(b => b.quantity === quantityCol);
    if (!block) return result;
    const col = block.quantity - 1;
    const label = asText(sheetCellValue(detection.sheet,detection.headerRow,col));
    if (!/цена|price/i.test(label)) return result;
    return [{ id:'block-price', column:col, label:`${label} (по блоку объема)`, price:budgetCellPrice(detection.sheet,cellKey(row,col)), quantityCol }];
  }
  const { maxColumn } = sheetBounds(detection.sheet);
  for (let col = 1; col <= maxColumn; col++) {
    const label = asText(sheetCellValue(detection.sheet, detection.headerRow, col));
    if (!/цена|price/i.test(label)) continue;
    const price = budgetCellPrice(detection.sheet, cellKey(row, col));
    result.push({ id: String(col), column:col, label, price, quantityCol });
  }
  return result;
}

export const christinaLinesCache = new WeakMap();

/** Read complete membership from the blank, including unmatched/zero-order goods. */
export function christinaProffLines(workbook, sheet) {
  if (!christinaLinesCache.has(workbook)) christinaLinesCache.set(workbook, new Map());
  const cache = christinaLinesCache.get(workbook);
  if (cache.has(sheet)) return cache.get(sheet);
  const styles = workbook.files['xl/styles.xml'];
  const styleDoc = styles ? XML_PARSER.parseFromString(strFromU8(styles), 'application/xml') : null;
  const xfRoot = styleDoc && firstElement(styleDoc.documentElement, 'cellXfs');
  const xfs = xfRoot ? elements(xfRoot, 'xf') : [];
  const fill = cell => Number(xfs[Number(cell?.node?.getAttribute('s') || 0)]?.getAttribute('fillId') || 0);
  const lines = [], byRow = new Map();
  let current = null;
  const lastRow = sheetBounds(sheet).maxRow;
  for (let row = 1; row <= lastRow; row++) {
    const article = asText(sheetCellValue(sheet, row, 1)).trim().toUpperCase();
    const name = asText(sheetCellValue(sheet, row, 2)).trim();
    if (!article && name && !/профессиональный уход/i.test(name)) {
      const fills = [1,2,3,4,5,6].map(col => fill(sheet.cells.get(cellKey(row,col))));
      // Adjacent cells can encode visually identical colors with distinct fill
      // IDs (theme vs RGB). A continuous filled band need not share one ID.
      if (fills.every(f => f > 1)) {
        if (/ОБЩАЯ ЛИНИЯ/i.test(name)) { current = null; continue; }
        const label = name.split(/\s+[–—-]\s+/)[0].trim();
        current = { id: label.toUpperCase(), name: label, required: [], members: [] };
        lines.push(current);
      }
    }
    if (!/^CHR\d+$/i.test(article) || !name) continue;
    // The approved September PROFF blank has no Nuance heading. Its product
    // names identify this separate line; never absorb it into preceding MUSE.
    if (/^Nuance\b/i.test(name) && current?.id !== 'NUANCE') {
      current = { id: 'NUANCE', name: 'Nuance', required: [], members: [] };
      lines.push(current);
    }
    if (!current || article === 'CHR133') continue;
    current.required.push(article);
    current.members.push({ row, article });
  }
  for (const line of lines) for (const member of line.members) byRow.set(member.row, {
    id: line.id, name: line.name, article: member.article, required: [...line.required],
  });
  cache.set(sheet, byRow);
  return byRow;
}

export function setBudgetFormula(sheet, row, col, formula, value) {
  const cell = findOrCreateCell(sheet, row, col);
  while (cell.firstChild) cell.removeChild(cell.firstChild);
  cell.removeAttribute('t');
  const f = sheet.xml.createElementNS(NS_MAIN, 'f');
  f.textContent = formula;
  const v = sheet.xml.createElementNS(NS_MAIN, 'v');
  v.textContent = String(value);
  cell.appendChild(f); cell.appendChild(v);
  sheet.cells.get(cellKey(row,col)).value = value;
}

export function budgetInvoicePriceColumn(sheet, headerRow, entry) {
  const qtyRef = `${columnNumberToName(entry.quantityCol)}${entry.blankRow}`;
  const priceColumns = [...sheet.cells.values()].filter(c => c.row === headerRow && /цена|price/i.test(asText(c.value))).map(c=>c.col);
  const referenced = new Set();
  for (const c of sheet.cells.values()) {
    if (c.row !== entry.blankRow) continue;
    const refs = budgetCellFormula(sheet,c).replace(/\$/g,'').match(/[A-Z]+\d+/g) || [];
    if (!refs.includes(qtyRef)) continue;
    for (const ref of refs) {
      const p = parseCellRef(ref);
      if (p.row === entry.blankRow && priceColumns.includes(p.col)) referenced.add(p.col);
    }
  }
  if (referenced.size === 1) return [...referenced][0];
  if (referenced.size > 1) throw new Error(`Несколько цен в формуле суммы: ${entry.name}. Нужно проверить бланк.`);
  const discounted = priceColumns.filter(col => /скид/i.test(asText(sheetCellValue(sheet,headerRow,col))));
  if (discounted.length === 1) return discounted[0];
  if (priceColumns.length === 1) return priceColumns[0];
  // Repeated SOTHYS blocks each use their own price immediately before quantity.
  if (entry.pricing.column === entry.quantityCol-1 && priceColumns.includes(entry.pricing.column)) return entry.pricing.column;
  throw new Error(`Не удалось определить цену для суммы: ${entry.name}. Формулы бланка не изменены.`);
}

/** Apply agreed prices to an export copy only. Repeated downloads never compound discounts. */
export function applyBudgetWorkbookPricing(workbook, sheetName, headerRow, entries = []) {
  if (!entries.length) return workbook;
  if (entries.some(e=>e.quantity>0 && !(e.price>0 && e.pricing?.column))) throw new Error('Не для всех заказанных позиций определена цена. Проверьте выбранную колонку цены.');
  // CHRISTINA supplier blanks keep club prices; procurement discounts live
  // only in the platform calculation, including equal PROFF complete sets.
  if (entries.every(e => e.preserveBlankPrices)) return workbook;
  const output = loadXlsx(saveXlsx(workbook));
  const sheet = output.sheets.find(s=>s.name===sheetName);
  if (!sheet) throw new Error('Не найден лист бланка для обновления скидки.');
  const prepared = entries.filter(e=>e.price>0 && e.pricing?.column).map(e=>({ ...e,
    effectiveCol: budgetInvoicePriceColumn(sheet,headerRow,e),
  }));
  const changes = new Map();
  for (const e of prepared) {
    const sourceRef = `${columnNumberToName(e.pricing.column)}${e.blankRow}`;
    let formula = budgetCellFormula(sheet,sheet.cells.get(cellKey(e.blankRow,e.effectiveCol))).replace(/\s|\$/g,'');
    formula = formula.replace(/^ROUND\((.*),2\)$/i,'$1');
    const direct = /^([A-Z]+\d+)\*\(1-([A-Z]+\d+)(\/100)?\)$/i.exec(formula);
    if (!direct || direct[1] !== sourceRef) continue;
    const address = parseCellRef(direct[2]);
    if (address.row >= headerRow) continue;
    const value = e.pricing.discount / (direct[3] ? 1 : 100);
    if (changes.has(direct[2]) && changes.get(direct[2]).value !== value) throw new Error('Разные скидки ссылаются на одну ячейку бланка.');
    changes.set(direct[2],{...address,value});
  }
  for (const change of changes.values()) setNumericCell(sheet,change.row,change.col,change.value);

  const fallback = prepared.filter(e=>Math.abs((budgetCellPrice(sheet,cellKey(e.blankRow,e.effectiveCol)) ?? -1)-e.price)>0.000001);
  if (fallback.length) {
    // Materialize shared formula followers before replacing any group master.
    const shared = [...sheet.cells.values()].filter(c=>firstElement(c.node,'f')?.getAttribute('t')==='shared')
      .map(c=>({c,formula:budgetCellFormula(sheet,c)}));
    for (const {c,formula} of shared) {
      if (!formula) throw new Error('Не удалось прочитать общую формулу цены Excel.');
      const f=firstElement(c.node,'f'); f.textContent=formula;
      for (const attr of ['t','si','ref']) f.removeAttribute(attr);
    }
    const bounds=sheetBounds(sheet);
    const baseCol=bounds.maxColumn+2, discountCol=baseCol+1;
    setTextCell(sheet,headerRow,baseCol,'Цена до нашей скидки');
    setTextCell(sheet,headerRow,discountCol,'Наша скидка, %');
    // Multiple blocks on the same row need independent bases and discount cells.
    const blocks=new Map();
    for (const e of fallback) {
      if (!blocks.has(e.quantityCol)) blocks.set(e.quantityCol,blocks.size);
      const offset=blocks.get(e.quantityCol)*2;
      const b=baseCol+offset,d=discountCol+offset;
      setTextCell(sheet,headerRow,b,'Цена до нашей скидки');
      setTextCell(sheet,headerRow,d,'Наша скидка, %');
      setNumericCell(sheet,e.blankRow,b,e.pricing.basePrice);
      setNumericCell(sheet,e.blankRow,d,e.pricing.discount);
      const formula=`ROUND(${columnNumberToName(b)}${e.blankRow}*(1-${columnNumberToName(d)}${e.blankRow}/100),2)`;
      setBudgetFormula(sheet,e.blankRow,e.effectiveCol,formula,e.price);
      if (e.effectiveCol===e.pricing.column) setTextCell(sheet,headerRow,e.effectiveCol,'Цена с учетом нашей скидки');
    }
    const dimension=firstElement(sheet.xml,'dimension');
    const updatedBounds=sheetBounds(sheet);
    if (dimension) dimension.setAttribute('ref',`A1:${columnNumberToName(updatedBounds.maxColumn)}${updatedBounds.maxRow}`);
  }
  // Refresh numeric caches as well as preserving Excel's automatic recalculation.
  for (const c of sheet.cells.values()) {
    if (!budgetCellFormula(sheet,c)) continue;
    const value=budgetCellPrice(sheet,cellKey(c.row,c.col));
    if (value==null || !Number.isFinite(value)) continue;
    let v=firstElement(c.node,'v');
    if (!v) {v=sheet.xml.createElementNS(NS_MAIN,'v');c.node.appendChild(v);}
    v.textContent=String(value); c.value=value;
  }
  for (const e of prepared) {
    const actual=budgetCellPrice(sheet,cellKey(e.blankRow,e.effectiveCol));
    if (actual==null || Math.abs(actual-e.price)>0.000001) throw new Error(`Цена в бланке не совпала с расчетом: ${e.name}`);
  }
  return output;
}
