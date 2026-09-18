import { normalizeOrderValue } from "./editRules.js";
import { baselineForReportRow } from "../report/reportModel.js";
import { rowKey } from "./reviewEdits.js";

function number(value) {
  const parsed = Number(String(value ?? "").replace(",", "."));
  return Number.isFinite(parsed) ? parsed : 0;
}

// budgetRequestRows maps report rows to the raw budget-plan API rows. Pricing,
// brand order rules and the CHRISTINA set discount run in calculation-service,
// so this only shapes data the browser already holds — no calculation here.
export function budgetRequestRows(rows, edits) {
  return rows
    .filter((row) => row.editable !== false && row.hasBudgetData)
    .map((row) => {
      const key = rowKey(row);
      const edit = edits.get(key) || {};
      const quantity = normalizeOrderValue(edit.value) ?? 0;
      const baseline = baselineForReportRow(row) ?? 0;
      return {
        key,
        name: row.blankName || row.sourceName || row.blankArticle || row.sourceArticle,
        category: row.budgetCategory,
        quantity,
        base_price: number(row.budgetPrice),
        demand: number(row.budgetDemand),
        stock: number(row.stock),
        transit: number(row.inTransit),
        outbound: 0,
        box_size: number(row.blankBoxSize),
        group: row.blankId || "main",
        line: row.christinaLine || null,
        locked: quantity !== baseline,
        excluded: false,
        unsafe: false,
      };
    });
}

// budgetPatches turns a plan preview into edit patches, keeping the manager's
// existing comment. The change note comes from the backend row.comment.
export function budgetPatches(plan, edits) {
  return plan.rows
    .filter((row) => row.quantity !== row.before)
    .map((row) => {
      const previous = edits.get(row.key) || { value: "", comment: "" };
      return {
        key: row.key,
        previous: { ...previous },
        next: {
          value: row.quantity,
          comment: appendBudgetComment(previous.comment, row.comment),
        },
      };
    });
}

export function appendBudgetComment(previous, note) {
  return [previous, note].filter(Boolean).join("; ");
}
