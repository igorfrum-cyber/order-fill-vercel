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
export function budgetGroup(row) {
 const label=String(row.blankLabel || "").toUpperCase();
 return row.christinaLine || label === "PROFF" ? "proff" : label === "HOME" ? "home" : row.blankId || "main";
}

function settingsFor(row, settings) {
 return typeof settings === "string" ? {priceColumn: settings} : settings[budgetGroup(row)] || {};
}

function selectedPrice(row, settings) {
 const {priceColumn = ""} = settingsFor(row,settings);
 if (priceColumn) return row.budgetPrices?.find((price) => String(price.column) === String(priceColumn));
 return row.budgetPrices?.find((price) => Number(price.price) === Number(row.budgetPrice));
}

export function budgetRequestRows(rows, edits, settings = "", lockedKeys = new Set()) {
 const editable = rows.filter((row) => row.editable !== false);
 const sourceCounts = new Map();
 for (const row of editable) if (row.sourceRow != null) sourceCounts.set(row.sourceRow,(sourceCounts.get(row.sourceRow) || 0)+1);
 return editable.map((row) => {
  const key = rowKey(row);
  const edit = edits.get(key) || {};
  const baseline = baselineForReportRow(row) ?? 0;
  const quantity = edits.has(key) ? normalizeOrderValue(edit.value) ?? 0 : normalizeOrderValue(row.inserted) ?? baseline;
  const config = settingsFor(row,settings);
  const discount = config.pricingMode === "gross" ? number(config.discount) : 0;
  return {
   key,
   name: row.blankName || row.sourceName || row.blankArticle || row.sourceArticle,
   category: row.budgetCategory,
   quantity,
   base_price: number(selectedPrice(row,settings)?.price ?? (config.priceColumn ? 0 : row.budgetPrice)),
   demand: number(row.budgetDemand),stock: number(row.stock),transit: number(row.inTransit),outbound: 0,
   box_size: number(row.blankBoxSize),group: budgetGroup(row),line: row.christinaLine || null,
   locked: lockedKeys.has(key) || (quantity !== baseline && edit.budgetApplied !== true),excluded: lockedKeys.has(key),
   unsafe: Boolean(row.duplicate || String(row.status || "").startsWith("warning") || (row.sourceRow != null && sourceCounts.get(row.sourceRow)>1)),
   ...(typeof settings === "object" ? {discount_override: discount} : {}),
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
          budgetApplied: true,
          comment: appendBudgetComment(previous.comment, row.comment),
        },
      };
    });
}

export function appendBudgetComment(previous, note) {
  return [...new Set([previous, note].filter(Boolean))].join("; ");
}

// Export pricing covers unchanged quantities too: agreeing a discount is a
// separate edit even when the budget planner keeps every quantity unchanged.
export function budgetPricingEntries(rows, plan, discount, settings = "") {
 const planned = new Map(plan.rows.map((row) => [row.key, row]));
 return rows.filter((row) => planned.has(rowKey(row))).map((row) => {
  const price = selectedPrice(row,settings);
  if (!price) return null;
  const result = planned.get(rowKey(row));
  return {row_key: rowKey(row),price_column: price.column,base_price: number(price.price),
   discount: number(result.discount ?? discount), price: number(result.price)};
 }).filter(Boolean);
}
