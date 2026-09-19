import { apiClient } from "./client.js";

// planOrderBudget asks calculation-service (via gateway) to replan supplier
// quantities to a target sum. All pricing and the CHRISTINA PROFF set discount
// run on the backend; the browser only sends raw report rows and renders.
export async function planOrderBudget(payload) {
  const data = await apiClient.request("/api/v1/order/budget-plan", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  return normalizeBudgetPlan(data);
}

export function normalizeBudgetPlan(data = {}) {
  const rows = (data.rows || []).map(mapPlannedRow);
  return {
    before: Number(data.before || 0),
    total: Number(data.total || 0),
    target: Number(data.target || 0),
    reason: data.reason || "",
    complete: Boolean(data.complete),
    christinaProffMode: data.christina_proff_mode || "standard",
    rows,
    lineSteps: (data.line_steps || []).map(mapLineStep),
    lineGroups: (data.line_groups || []).map((group) => ({
      id: group.id,
      name: group.name,
      valid: Boolean(group.valid),
      sets: group.sets,
      saving: Number(group.saving || 0),
      net: Number(group.net || 0),
    })),
    fastTotal: Number(data.fast_total || 0),
    fastComplete: Boolean(data.fast_complete),
    fastReason: data.fast_reason || "",
    fastRows: (data.fast_rows || []).map(mapPlannedRow),
    fastLineSteps: (data.fast_line_steps || []).map(mapLineStep),
    compare: data.compare ? mapCompare(data.compare) : null,
  };
}

function mapPlannedRow(row) {
  return {
    key: row.key,
    name: row.name,
    category: row.category,
    before: Number(row.before || 0),
    quantity: Number(row.quantity || 0),
    comment: row.comment || "",
  };
}

function mapLineStep(step) {
  return {
    id: step.id,
    name: step.name,
    sets: step.sets,
    added: Number(step.added || 0),
    saved: Number(step.saved || 0),
  };
}

function mapCompare(compare) {
  return {
    match: Boolean(compare.match),
    standardMs: Number(compare.standard_ms || 0),
    fastMs: Number(compare.fast_ms || 0),
    mismatches: (compare.mismatches || []).map((item) => ({
      where: item.where || "",
      key: item.key || "",
      field: item.field || "",
      want: item.want ?? item.want_cents ?? "",
      got: item.got ?? item.got_cents ?? "",
    })),
  };
}

export function christinaStandardJSON(plan) {
  return {
    before: plan.before,
    total: plan.total,
    complete: plan.complete,
    reason: plan.reason,
    line_steps: plan.lineSteps,
    rows: plan.rows,
  };
}

export function christinaFastJSON(plan) {
  return {
    fast_total: plan.fastTotal,
    fast_complete: plan.fastComplete,
    fast_reason: plan.fastReason,
    fast_line_steps: plan.fastLineSteps,
    fast_rows: plan.fastRows,
  };
}

export function christinaMismatchJSON(plan) {
  return {
    total: plan.total,
    fast_total: plan.fastTotal,
    mismatches: plan.compare?.mismatches || [],
  };
}
