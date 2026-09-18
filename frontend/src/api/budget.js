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

function normalizeBudgetPlan(data = {}) {
  return {
    before: Number(data.before || 0),
    total: Number(data.total || 0),
    target: Number(data.target || 0),
    reason: data.reason || "",
    complete: Boolean(data.complete),
    rows: (data.rows || []).map((row) => ({
      key: row.key,
      name: row.name,
      category: row.category,
      before: Number(row.before || 0),
      quantity: Number(row.quantity || 0),
      comment: row.comment || "",
    })),
    lineSteps: (data.line_steps || []).map((step) => ({
      id: step.id,
      name: step.name,
      sets: step.sets,
      added: Number(step.added || 0),
      saved: Number(step.saved || 0),
    })),
    lineGroups: (data.line_groups || []).map((group) => ({
      id: group.id,
      name: group.name,
      valid: Boolean(group.valid),
      sets: group.sets,
      saving: Number(group.saving || 0),
      net: Number(group.net || 0),
    })),
  };
}
