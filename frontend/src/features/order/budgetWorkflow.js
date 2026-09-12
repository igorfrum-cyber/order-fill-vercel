import { appendBudgetComment, budgetChangeComment, budgetOrderRules } from "./budgetPlanner.js";
import { normalizeOrderValue } from "./editRules.js";
import { baselineForReportRow } from "../report/reportModel.js";
import { rowKey } from "./reviewEdits.js";

function number(value) {
  const parsed = Number(String(value ?? "").replace(",", "."));
  return Number.isFinite(parsed) ? parsed : 0;
}

export function budgetRowsFromReport(rows, edits, { brand, deliveryWeeks } = {}) {
  return rows.filter((row) => row.editable !== false && row.hasBudgetData).map((row) => {
    const key = rowKey(row);
    const edit = edits.get(key) || {};
    const quantity = normalizeOrderValue(edit.value) ?? 0;
    const baseline = baselineForReportRow(row) ?? 0;
    const rules = budgetOrderRules(brand, row.blankName || row.sourceName, row.blankBoxSize);
    return {
      key,
      name: row.blankName || row.sourceName || row.blankArticle || row.sourceArticle,
      category: row.budgetCategory,
      quantity,
      price: number(row.budgetPrice),
      demand: number(row.budgetDemand),
      delivery: number(deliveryWeeks) * 0.25,
      stock: number(row.stock),
      transit: number(row.inTransit),
      outbound: 0,
      ...rules,
      locked: quantity !== baseline,
      excluded: false,
      unsafe: false,
    };
  });
}

export function budgetPatches(plan, edits) {
  return plan.rows.filter((row) => row.quantity !== row.before).map((row) => {
    const previous = edits.get(row.key) || { value: "", comment: "" };
    return {
      key: row.key,
      previous: { ...previous },
      next: {
        value: row.quantity,
        comment: appendBudgetComment(previous.comment, budgetChangeComment(row)),
      },
    };
  });
}
