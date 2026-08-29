import test from "node:test";
import assert from "node:assert/strict";

import { qualityWarningSummary } from "./qualityWarnings.js";

test("qualityWarningSummary counts issues that need a download confirmation", () => {
  const summary = qualityWarningSummary({
    rows: [
      { key: "a", status: "warning_name_differs", editable: true, recommended: 10, rounded: 10, inserted: 10 },
      { key: "b", status: "not_in_source", editable: true, recommended: 0, rounded: 0, inserted: null },
      { key: "c", status: "not_in_blank", editable: false },
      { key: "d", status: "matched", duplicate: true, editable: true, recommended: 6, rounded: 6, inserted: 6 },
    ],
    results: [{ summary: { blankDuplicateArticles: 2 } }],
    edits: new Map(),
  });

  assert.equal(summary.issueCount, 1);
  assert.equal(summary.duplicateCount, 1);
  assert.equal(summary.notInSourceCount, 1);
  assert.equal(summary.notInBlankCount, 1);
  assert.equal(summary.blankDuplicateCount, 2);
  assert.equal(summary.manualCount, 0);
  assert.equal(summary.total, 6);
});
