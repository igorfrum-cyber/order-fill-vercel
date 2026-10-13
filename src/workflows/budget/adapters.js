import { parseNumber } from "./../../shared/text.js";
import { budgetPrices, christinaProffLines } from "./../../infrastructure/excel/pricing.js";
import { budgetOrderRules } from "./../../domain/order/budget-rules.js";

export function budgetReportRows(result, brand) {
  const lines = brand === 'christina' && result.blankId === 'proff'
    ? christinaProffLines(result.blankWorkbook, result.blankDetection.sheet) : new Map();
  return result.reportRows.filter(r => r.editable !== false).map(r => ({
    ...r.budget, ...budgetOrderRules(brand, r.blankName, r.blankBoxSize),
    key: r.key, name: r.blankName, group: result.blankId,
    blankRow: r.blankRow, quantityCol: r.blankQuantityCol,
    inventoryKey: r.sourceRow == null ? `unmatched:${r.key}` : String(r.sourceRow),
    unsafe: Boolean(r.duplicate || r.status.startsWith('warning')),
    stock: parseNumber(r.stock) || 0, transit: parseNumber(r.inTransit) || 0,
    prices: budgetPrices(result.blankDetection, r.blankRow, r.blankQuantityCol),
    line: lines.get(r.blankRow), preserveBlankPrices: brand === 'christina',
  }));
}
