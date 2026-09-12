import test from "node:test";
import assert from "node:assert/strict";

import { budgetPatches, budgetRowsFromReport } from "./budgetWorkflow.js";

test("budgetRowsFromReport uses the current edit and locks a manual deviation", () => {
  const rows = [{
    key: "row-1", editable: true, hasBudgetData: true, inserted: 3,
    blankName: "Крем", blankBoxSize: "3", budgetCategory: "B", budgetDemand: 10,
    budgetPrice: 100, stock: "2", inTransit: "1",
  }];
  const input = budgetRowsFromReport(rows, new Map([["row-1", { value: 6, comment: "решение" }]]), {
    brand: "angiopharm", deliveryWeeks: 1,
  });
  assert.deepEqual(input[0], {
    key: "row-1", name: "Крем", category: "B", quantity: 6, price: 100,
    demand: 10, delivery: 0.25, stock: 2, transit: 1, outbound: 0,
    unit: 1, step: 3, minimum: 3, locked: true, excluded: false, unsafe: false,
  });
});

test("budgetPatches preserves the previous comment and returns an undo snapshot", () => {
  const edits = new Map([["row-1", { value: 3, comment: "проверено" }]]);
  const patches = budgetPatches({ rows: [{ key: "row-1", name: "Крем", unit: 1, before: 3, quantity: 6 }] }, edits);
  assert.equal(patches[0].next.value, 6);
  assert.match(patches[0].next.comment, /^проверено; Добавилось 3 шт\./);
  assert.deepEqual(patches[0].previous, { value: 3, comment: "проверено" });
});
