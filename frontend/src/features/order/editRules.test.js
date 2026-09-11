import test from "node:test";
import assert from "node:assert/strict";

import { editRequiresComment, normalizeOrderValue, parseQuantityInput } from "./editRules.js";

test("normalizeOrderValue accepts empty and comma decimal values", () => {
  assert.equal(normalizeOrderValue(""), null);
  assert.equal(normalizeOrderValue("12,5"), 12.5);
});

test("parseQuantityInput keeps decimals and a trailing separator while typing", () => {
  assert.equal(parseQuantityInput(""), "");
  assert.equal(parseQuantityInput("12,5"), 12.5);
  assert.equal(parseQuantityInput("12.5"), 12.5);
  assert.equal(parseQuantityInput("12."), "12.");
});

test("editRequiresComment ignores unchanged auto-commented value", () => {
  assert.equal(editRequiresComment({
    value: 12,
    baseline: 10,
    initial: 12,
    comment: "до коробки",
    autoComment: "до коробки",
  }), false);
});

test("editRequiresComment requires a new comment when value differs from baseline", () => {
  assert.equal(editRequiresComment({
    value: 12,
    baseline: 10,
    initial: 10,
    comment: "",
    autoComment: "",
  }), true);
});
