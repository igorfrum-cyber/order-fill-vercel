import test from "node:test";
import assert from "node:assert/strict";

import { collectReviewEdits, commentGateRows, hasManualDeviations, patchEdit, validateReviewEdits } from "./reviewEdits.js";

function editMap(entries) {
  return new Map(entries);
}

test("validateReviewEdits flags rows where the inserted value changed without a new comment", () => {
  const rows = [
    { key: "a", editable: true, inserted: 10, recommended: 10, rounded: 10, autoComment: "" },
    { key: "b", editable: true, inserted: 12, recommended: 10, rounded: 10, autoComment: "до коробки" },
  ];
  const edits = editMap([
    ["a", { value: "18", comment: "" }],
    ["b", { value: "12", comment: "до коробки" }],
  ]);

  const invalid = validateReviewEdits(rows, edits);
  assert.deepEqual(invalid, ["a"]);
});

test("commentGateRows lists only rows that still need a comment after a quantity change", () => {
  const rows = [
    { key: "a", editable: true, inserted: 10, recommended: 10, rounded: 10, blankArticle: "CT02", blankName: "Крем" },
    { key: "b", editable: true, inserted: 12, recommended: 10, rounded: 10, autoComment: "до коробки", blankArticle: "AA04", blankName: "Сыворотка" },
  ];
  const edits = editMap([
    ["a", { value: "18", comment: "" }],
    ["b", { value: "12", comment: "до коробки" }],
  ]);
  assert.deepEqual(commentGateRows(rows, edits).map((row) => row.key), ["a"]);
});

test("commentGateRows clears once a new comment is written", () => {
  const rows = [
    { key: "a", editable: true, inserted: 10, recommended: 10, rounded: 10, blankArticle: "CT02", blankName: "Крем" },
  ];
  const edits = editMap([["a", { value: "18", comment: "договорились с поставщиком" }]]);
  assert.deepEqual(commentGateRows(rows, edits), []);
});

test("collectReviewEdits sends only editable rows", () => {
  const rows = [
    { key: "a", blankId: "main", blankRow: 4, editable: true },
    { key: "b", blankId: "main", blankRow: 5, editable: false },
  ];
  const edits = editMap([
    ["a", { value: "6", comment: "меньше склада" }],
  ]);

  assert.deepEqual(collectReviewEdits(rows, edits), [
    { key: "a", blankId: "main", blankRow: 4, value: "6", comment: "меньше склада" },
  ]);
});

test("hasManualDeviations is false when the reviewer kept a box-adjusted quantity", () => {
  const rows = [
    { key: "a", editable: true, inserted: 12, recommended: 10, rounded: 10, autoComment: "до коробки" },
  ];
  const edits = editMap([["a", { value: "12", comment: "до коробки" }]]);
  assert.equal(hasManualDeviations(rows, edits), false);
});

test("hasManualDeviations is true when a quantity left the engine baseline", () => {
  const rows = [
    { key: "a", editable: true, inserted: 12, recommended: 10, rounded: 12, autoComment: "до коробки" },
  ];
  const edits = editMap([["a", { value: "18", comment: "вручную" }]]);
  assert.equal(hasManualDeviations(rows, edits), true);
});

test("hasManualDeviations is true when the reviewer kept the quantity but typed a new comment", () => {
  const rows = [
    { key: "a", editable: true, inserted: 10, recommended: 10, rounded: 10, autoComment: "" },
  ];
  const edits = editMap([["a", { value: "10", comment: "подтвердили заказ" }]]);
  assert.equal(hasManualDeviations(rows, edits), true);
});

test("patchEdit merges a field without dropping the other", () => {
  const edits = editMap([["a", { value: 10, comment: "" }]]);
  const afterQuantity = patchEdit(edits, "a", { value: 18 });
  assert.deepEqual(afterQuantity.get("a"), { value: 18, comment: "" });
  const afterComment = patchEdit(afterQuantity, "a", { comment: "договорились" });
  assert.deepEqual(afterComment.get("a"), { value: 18, comment: "договорились" });
});
