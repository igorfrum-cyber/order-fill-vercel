import test from "node:test";
import assert from "node:assert/strict";

import { collectReviewEdits, hasManualDeviations, validateReviewEdits } from "./reviewEdits.js";

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

test("hasManualDeviations is false when the reviewer kept the engine quantities", () => {
  const rows = [
    { key: "a", editable: true, inserted: 12, recommended: 10, rounded: 12 },
  ];
  const edits = editMap([["a", { value: "12", comment: "до коробки" }]]);
  assert.equal(hasManualDeviations(rows, edits), false);
});

test("hasManualDeviations is true when a quantity left the engine baseline", () => {
  const rows = [
    { key: "a", editable: true, inserted: 12, recommended: 10, rounded: 12 },
  ];
  const edits = editMap([["a", { value: "18", comment: "вручную" }]]);
  assert.equal(hasManualDeviations(rows, edits), true);
});
