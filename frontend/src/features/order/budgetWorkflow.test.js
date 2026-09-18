import test from "node:test";
import assert from "node:assert/strict";

import { appendBudgetComment, budgetPatches, budgetRequestRows } from "./budgetWorkflow.js";

test("budgetRequestRows sends raw report numbers and locks a manual deviation", () => {
  const rows = [{
    key: "row-1", editable: true, hasBudgetData: true, inserted: 3,
    blankName: "Крем", blankBoxSize: "3", blankId: "proff", budgetCategory: "B", budgetDemand: 10,
    budgetPrice: 100, stock: "2", inTransit: "1", christinaLine: { id: "MUSE", name: "MUSE", article: "0", required: ["0"] },
  }];
  const input = budgetRequestRows(rows, new Map([["row-1", { value: 6, comment: "решение" }]]));
  assert.deepEqual(input[0], {
    key: "row-1", name: "Крем", category: "B", quantity: 6, base_price: 100,
    demand: 10, stock: 2, transit: 1, outbound: 0, box_size: 3,
    group: "proff", line: { id: "MUSE", name: "MUSE", article: "0", required: ["0"] },
    locked: true, excluded: false, unsafe: false,
  });
});

test("budgetRequestRows carries the base price without discounting it", () => {
  const rows = [{
    key: "row-1", editable: true, hasBudgetData: true, inserted: 2,
    blankName: "Крем", budgetCategory: "B", budgetDemand: 10, budgetPrice: 199.99,
  }];
  const [row] = budgetRequestRows(rows, new Map());
  assert.equal(row.base_price, 199.99);
  assert.equal(row.group, "main");
  assert.equal(row.line, null);
});

test("budgetPatches keeps the previous comment and takes the note from the plan row", () => {
  const edits = new Map([["row-1", { value: 3, comment: "проверено" }]]);
  const plan = { rows: [{ key: "row-1", name: "Крем", before: 3, quantity: 6, comment: "Добавилось 3 шт. Для закупа до суммы." }] };
  const patches = budgetPatches(plan, edits);
  assert.equal(patches[0].next.value, 6);
  assert.match(patches[0].next.comment, /^проверено; Добавилось 3 шт\./);
  assert.deepEqual(patches[0].previous, { value: 3, comment: "проверено" });
});

test("appendBudgetComment joins non-empty parts", () => {
  assert.equal(appendBudgetComment("ручная правка", "Уменьшено"), "ручная правка; Уменьшено");
  assert.equal(appendBudgetComment("", "Уменьшено"), "Уменьшено");
  assert.equal(appendBudgetComment("ручная правка", ""), "ручная правка");
});
