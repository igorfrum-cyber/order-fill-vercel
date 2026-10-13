import { loadXlsx, saveXlsx } from "./../../infrastructure/excel/archive.js";
import { normalizeName } from "./../../domain/matching/identity.js";
import { detectCalculationColumns, detectColumns } from "./../../infrastructure/excel/columns.js";
import { readSource } from "./../order/read-source.js";
import { mergeTyumenSources } from "./../tyumen/merge.js";
import { brandRule } from "./../../domain/brands/rules.js";

/** Standalone recalculation shares the source pipeline, not supplier rounding. */
export function recalculateOrderTable({workbook,warehouseWorkbook,brand,orderMonth,fileName='',warehouseFileName=''}) {
  const output=warehouseWorkbook ? mergeTyumenSources({officeWorkbook:workbook,warehouseWorkbook,brand,officeFileName:fileName,warehouseFileName,preserveFacts:true}) : loadXlsx(saveXlsx(workbook));
  detectCalculationColumns(detectColumns(output,'source'));
  const result=readSource(output,orderMonth,brandRule(brand),fileName);
  const counts=new Map();
  for(const item of result.items){const key=item.article || normalizeName(item.name);counts.set(key,(counts.get(key)||0)+1);}
  return {workbook:output,rows:result.items.map(item=>({...item,review:counts.get(item.article || normalizeName(item.name))>1 || item.sourceComment.includes('Проверить:')}))};
}
