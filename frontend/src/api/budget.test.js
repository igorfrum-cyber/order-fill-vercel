import test from "node:test";
import assert from "node:assert/strict";

import { planOrderBudget } from "./budget.js";
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
    rows: [{ key: "MUSE:3", name: "MUSE 3", category: "A+", before: 3, quantity: 6, comment: "Добавилось 3 шт. Для закупа до суммы." }],
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
