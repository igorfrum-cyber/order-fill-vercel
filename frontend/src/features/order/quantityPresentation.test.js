import test from "node:test";
import assert from "node:assert/strict";

import { quantityDivergesFromRecommendation, roundingComment } from "./quantityPresentation.js";

test("quantityDivergesFromRecommendation highlights box rounding against the raw recommendation", () => {
  const row = { recommended: 10, inserted: 12, rounded: 12, autoComment: "до коробки", boxAdjusted: true };
  assert.equal(quantityDivergesFromRecommendation(row, 12), true);
  assert.equal(roundingComment(row), "до коробки");
});

test("quantityDivergesFromRecommendation is quiet when inserted matches recommendation", () => {
  const row = { recommended: 12, inserted: 12, rounded: 12, autoComment: "" };
  assert.equal(quantityDivergesFromRecommendation(row, 12), false);
  assert.equal(roundingComment(row), "");
});

test("quantityDivergesFromRecommendation follows the live edit value", () => {
  const row = { recommended: 12, inserted: 12, rounded: 12, autoComment: "" };
  assert.equal(quantityDivergesFromRecommendation(row, 18), true);
});
