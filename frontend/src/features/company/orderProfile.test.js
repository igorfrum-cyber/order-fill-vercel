import test from "node:test";
import assert from "node:assert/strict";
import { discountText, orderProfileIssue, parseDiscount, profileForSave, normalizeOrderProfile } from "./orderProfile.js";

test("discount accepts a Russian decimal and keeps exact basis points", () => {
  assert.deepEqual(parseDiscount("30,25"), { set: true, basisPoints: 3025, issue: "" });
  assert.equal(discountText({ discount_set: true, discount_basis_points: 3025 }), "30,25");
});

test("discount rejects values outside 0..100", () => {
  assert.match(parseDiscount("100,01").issue, /от 0 до 100/);
  assert.match(parseDiscount("3.141").issue, /двух знаков/);
});

test("profile validates phone and serializes an explicit zero discount", () => {
  const profile = normalizeOrderProfile({ contact_phone: "abc" });
  assert.match(orderProfileIssue(profile, { angiopharm: "", skin_synergy: "", klapp: "" }), /телефон/i);
  const saved = profileForSave(normalizeOrderProfile({}), { angiopharm: "0", skin_synergy: "", klapp: "" });
  assert.deepEqual(saved.brand_terms[0], {
    brand: "angiopharm",
    dealer_name: "",
    payment_method: "",
    payment_control: "",
    customer_type: "",
    discount_set: true,
    discount_basis_points: 0,
  });
});
