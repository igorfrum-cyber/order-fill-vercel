import test from "node:test";
import assert from "node:assert/strict";

import {
  christinaFastJSON,
  christinaMismatchJSON,
  christinaStandardJSON,
  normalizeBudgetPlan,
  planOrderBudget,
} from "./budget.js";
import { apiClient } from "./client.js";

function stubClient(json) {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = async (url, options) => {
    calls.push({ url, options });
    return {
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => json,
    };
  };
  return {
    calls,
    restore() {
      apiClient.fetcher = originalFetcher;
      apiClient.baseUrl = originalBase;
    },
  };
}

test("planOrderBudget posts the payload and normalizes the response", async () => {
  const stub = stubClient({
    before: 5640,
    total: 5880,
    target: 6100,
    reason: "",
    complete: true,
    rows: [{ key: "MUSE:3", name: "MUSE 3", category: "A+", before: 3, quantity: 6, price:70, discount:30,comment: "Добавилось 3 шт. Для закупа до суммы." }],
    line_steps: [{ id: "MUSE", name: "MUSE", sets: 6, added: 300, saved: 60 }],
    line_groups: [{ id: "MUSE", name: "MUSE", valid: true, sets: 6, saving: 120, net: 5880 }],
  });
  try {
    const payload = { brand: "christina", target: 6100, discount: 0, delivery_weeks: 1, rows: [{ key: "MUSE:3" }] };
    const plan = await planOrderBudget(payload);
    assert.equal(stub.calls[0].url, "/api/v1/order/budget-plan");
    assert.equal(stub.calls[0].options.method, "POST");
    assert.deepEqual(JSON.parse(stub.calls[0].options.body), payload);
    assert.equal(plan.complete, true);
    assert.equal(plan.total, 5880);
    assert.equal(plan.rows[0].category, "A+");
    assert.equal(plan.rows[0].price, 70);
    assert.equal(plan.rows[0].discount, 30);
    assert.equal(plan.lineSteps[0].saved, 60);
    assert.equal(plan.lineGroups[0].net, 5880);
  } finally {
    stub.restore();
  }
});

test("planOrderBudget tolerates a sparse response", async () => {
  const stub = stubClient({ before: 0, total: 0, target: 10, complete: false });
  try {
    const plan = await planOrderBudget({ target: 10, rows: [] });
    assert.deepEqual(plan.rows, []);
    assert.deepEqual(plan.lineSteps, []);
    assert.deepEqual(plan.lineGroups, []);
    assert.equal(plan.reason, "");
  } finally {
    stub.restore();
  }
});

test("normalizeBudgetPlan maps fast, Christina, comparison and mismatch fields", () => {
  const plan = normalizeBudgetPlan({
    christina_proff_mode: "compare",
    rows: [{ key: "r1", name: "Row", category: "A", before: "2", quantity: "3", comment: "note" }],
    line_steps: [{ id: "l1", name: "Line", sets: 2, added: "5", saved: "1" }],
    line_groups: [{ id: "g1", name: "Group", valid: true, sets: 2, saving: "4", net: "20" }],
    fast_total: "30",
    fast_complete: true,
    fast_reason: "fast reason",
    fast_rows: [{ key: "r2", name: "Fast row", category: "B", before: "1", quantity: "4" }],
    fast_line_steps: [{ id: "l2", name: "Fast line", sets: 3, added: "6", saved: "2" }],
    compare: {
      match: false,
      standard_ms: "10",
      fast_ms: "5",
      mismatches: [
        { where: "row", key: "r1", field: "quantity", want: "3", got: "4" },
        { want_cents: 1200, got_cents: 1300 },
      ],
    },
  });

  assert.equal(plan.christinaProffMode, "compare");
  assert.equal(plan.rows[0].quantity, 3);
  assert.equal(plan.rows[0].comment, "note");
  assert.deepEqual(plan.lineSteps[0], { id: "l1", name: "Line", sets: 2, added: 5, saved: 1 });
  assert.deepEqual(plan.lineGroups[0], { id: "g1", name: "Group", valid: true, sets: 2, saving: 4, net: 20 });
  assert.equal(plan.fastTotal, 30);
  assert.equal(plan.fastComplete, true);
  assert.equal(plan.fastReason, "fast reason");
  assert.equal(plan.fastRows[0].quantity, 4);
  assert.equal(plan.fastLineSteps[0].added, 6);
  assert.deepEqual(plan.compare, {
    match: false,
    standardMs: 10,
    fastMs: 5,
    mismatches: [
      { where: "row", key: "r1", field: "quantity", want: "3", got: "4" },
      { where: "", key: "", field: "", want: 1200, got: 1300 },
    ],
  });
});

test("Christina JSON helpers expose only the selected plan data", () => {
  const plan = normalizeBudgetPlan({
    before: 10,
    total: 12,
    complete: true,
    reason: "standard",
    rows: [{ key: "r1" }],
    line_steps: [{ id: "s1" }],
    fast_total: 15,
    fast_complete: false,
    fast_reason: "fast",
    fast_rows: [{ key: "r2" }],
    fast_line_steps: [{ id: "s2" }],
    compare: { mismatches: [{ key: "r3" }] },
  });

  assert.deepEqual(christinaStandardJSON(plan), {
    before: 10,
    total: 12,
    complete: true,
    reason: "standard",
    line_steps: plan.lineSteps,
    rows: plan.rows,
  });
  assert.deepEqual(christinaFastJSON(plan), {
    fast_total: 15,
    fast_complete: false,
    fast_reason: "fast",
    fast_line_steps: plan.fastLineSteps,
    fast_rows: plan.fastRows,
  });
  assert.deepEqual(christinaMismatchJSON(plan), {
    total: 12,
    fast_total: 15,
    mismatches: plan.compare.mismatches,
  });
  assert.deepEqual(christinaMismatchJSON(normalizeBudgetPlan()), {
    total: 0,
    fast_total: 0,
    mismatches: [],
  });
});
