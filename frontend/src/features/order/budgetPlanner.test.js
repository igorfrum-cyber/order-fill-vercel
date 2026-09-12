import test from "node:test";
import assert from "node:assert/strict";

import { appendBudgetComment, budgetChangeComment, budgetOrderRules, coverage, discountValue, planBudget } from "./budgetPlanner.js";

const row = (key, category = "A", extra = {}) => ({
  key, name: key, category, quantity: 10, demand: 10, stock: 0, transit: 0, delivery: 0.25, price: 100, unit: 1, step: 1, minimum: 1, ...extra,
});

test("discountValue accepts percent and rejects 100", () => {
  assert.equal(discountValue("30%"), 30);
  assert.equal(discountValue("30"), 30);
  assert.throws(() => discountValue("100%"));
});

test("budgetChangeComment matches origin/main copy", () => {
  assert.equal(budgetChangeComment({ before: 13, quantity: 4, unit: 1 }), "Уменьшено на 9 шт. Для снижения заказа до указанной суммы.");
  assert.equal(budgetChangeComment({ before: 3, quantity: 0, unit: 5 }), "Уменьшено на 3 уп. Для снижения заказа до указанной суммы.");
  assert.equal(budgetChangeComment({ before: 0, quantity: 0, unit: 1 }), "");
  assert.equal(budgetChangeComment({ quantity: 4, unit: 1 }), "");
  assert.equal(budgetChangeComment({ before: 4, quantity: 13, unit: 1 }), "Добавилось 9 шт. Для закупа до суммы.");
});

test("appendBudgetComment preserves the existing separator", () => {
  assert.equal(appendBudgetComment("Ручная правка", "Уменьшено"), "Ручная правка; Уменьшено");
  assert.equal(appendBudgetComment("Ручная правка", ""), "Ручная правка");
  assert.equal(appendBudgetComment("", "Уменьшено"), "Уменьшено");
});

test("planBudget stages permissions and does not mutate input", () => {
  let p = planBudget([row("a"), row("c", "C")], 2200);
  assert.equal(p.rows[0].quantity, 10);
  assert.equal(p.rows[1].quantity, 12);
  p = planBudget([row("manual", "A", { quantity: 15 })], 1700);
  assert.equal(p.rows[0].before, 15);
  assert.equal(p.rows[0].quantity, 17);
  p = planBudget([row("locked", "C", { locked: true }), row("free")], 2200);
  assert.equal(p.rows[0].quantity, 10);
  assert.equal(p.rows[1].quantity, 12);
  p = planBudget([row("none", "C", { demand: 0 }), row("free")], 2200);
  assert.equal(p.rows[0].quantity, 10);
  p = planBudget([row("zero", "C", { quantity: 0 }), row("manual-zero", "C", { quantity: 0, excluded: true })], 200);
  assert.equal(p.rows[0].quantity, 2);
  assert.equal(p.rows[1].quantity, 0);
  p = planBudget([row("cap", "A", { quantity: 60 })], 6200);
  assert.equal(p.reason, "overSix");
  p = planBudget([row("cap", "A", { quantity: 60 })], 6200, { allowOverSix: true });
  assert.equal(p.total, 6200);
  p = planBudget([row("floor", "A", { quantity: 15 })], 1200);
  assert.equal(p.reason, "belowOne");
  assert.equal(p.total, 1300);
  p = planBudget([row("floor", "A", { quantity: 15 })], 1200, { allowBelowOne: true });
  assert.equal(p.total, 1200);
  p = planBudget([row("c", "C", { quantity: 10 }), row("a", "A", { quantity: 10 })], 1800, { allowBelowOne: true });
  assert.equal(p.rows[0].quantity, 8);
  assert.equal(p.rows[1].quantity, 10);
});

test("planBudget units and isolation", () => {
  const nova = budgetOrderRules("novacutan", "Novacutan SBIO, 2 мл");
  assert.deepEqual(nova, { unit: 1, step: 10, minimum: 100 });
  let p = planBudget([row("nova", "A", { ...nova, quantity: 0, demand: 100 })], 9800);
  assert.equal(p.total, 10000);
  assert.ok(p.complete);
  p = planBudget([row("nova", "A", { ...nova, quantity: 0, demand: 100 })], 9000);
  assert.ok(!p.complete);
  const mask = budgetOrderRules("novacutan", "EYE FILLER MASK NOVACUTAN");
  assert.equal(mask.unit, 5);
  assert.equal(coverage(row("mask", "A", { ...mask, demand: 10 }), 12), 6);
  for (const [brand, box, step] of [["christina", null, 3], ["klapp", null, 3], ["angiopharm", 6, 6], ["levissime", 4, 4], ["skin_synergy", null, 1], ["sothys", null, 1]]) {
    const rules = budgetOrderRules(brand, "Товар", box);
    assert.equal(rules.step, step, brand);
    const planned = planBudget([row(brand, "A", { ...rules, quantity: 0, demand: 100 })], step * 100);
    assert.equal(planned.rows[0].quantity, step, brand);
    assert.ok(planned.complete, brand);
  }
  for (let i = 1; i < 80; i++) {
    const source = [row("a", "A", { quantity: i, price: 37 }), row("b", "B", { quantity: 20, price: 59 })];
    const result = planBudget(source, 3000, { allowOverSix: true, allowBelowOne: true });
    if (result.complete) assert.ok(result.total >= 3000 && result.total <= 3150 || result.before >= 3000 && result.total <= 3000);
    assert.deepEqual(source[0].quantity, i);
  }
});
