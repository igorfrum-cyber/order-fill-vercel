import test from "node:test";
import assert from "node:assert/strict";
import { discountText, orderProfileIssue, parseDiscount, profileForSave, normalizeOrderProfile, setTerms, termsFor } from "./orderProfile.js";

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

test("profile normalization clones brand terms and fills missing fields", () => {
  const sourceTerms = [{ brand: "klapp", discount_set: true, discount_basis_points: 1200 }];
  const normalized = normalizeOrderProfile({ brand_terms: sourceTerms, legal_name: "Acme" });
  assert.equal(normalized.legal_name, "Acme");
  assert.equal(normalized.address, "");
  assert.deepEqual(normalized.brand_terms, sourceTerms);
  assert.notEqual(normalized.brand_terms[0], sourceTerms[0]);
  assert.deepEqual(normalizeOrderProfile({ brand_terms: "not-an-array" }).brand_terms, []);
  assert.deepEqual(normalizeOrderProfile(null).brand_terms, []);
});

test("termsFor returns existing brand data or a complete default", () => {
  const profile = { brand_terms: [{ brand: "klapp", dealer_name: "Dealer" }] };
  assert.deepEqual(termsFor(profile, "klapp"), { brand: "klapp", dealer_name: "Dealer" });
  assert.deepEqual(termsFor(profile, "angiopharm"), {
    brand: "angiopharm",
    dealer_name: "",
    payment_method: "",
    payment_control: "",
    customer_type: "",
    discount_basis_points: 0,
    discount_set: false,
  });
});

test("setTerms replaces selected terms, appends new brands and retains other brands", () => {
  const original = {
    legal_name: "Acme",
    brand_terms: [
      { brand: "klapp", dealer_name: "old" },
      { brand: "skin_synergy", dealer_name: "keep" },
      { brand: "klapp", dealer_name: "duplicate" },
    ],
  };
  const updated = setTerms(original, "klapp", { dealer_name: "new" });
  assert.equal(updated.legal_name, "Acme");
  assert.deepEqual(updated.brand_terms, [
    { brand: "skin_synergy", dealer_name: "keep" },
    { brand: "klapp", dealer_name: "new" },
  ]);
  assert.equal(original.brand_terms.length, 3);
  const added = setTerms({ brand_terms: [] }, "angiopharm", { discount_set: true });
  assert.equal(added.brand_terms[0].brand, "angiopharm");
  assert.equal(added.brand_terms[0].discount_set, true);
});

test("discount text and parsing cover unset, empty, malformed and percentage bounds", () => {
  assert.equal(discountText({ discount_set: false, discount_basis_points: 3000 }), "");
  assert.equal(discountText({ discount_set: true, discount_basis_points: "bad" }), "0");
  assert.deepEqual(parseDiscount(null), { set: false, basisPoints: 0, issue: "" });
  assert.match(parseDiscount("-1").issue, /не более двух знаков/);
  assert.match(parseDiscount("101").issue, /от 0 до 100%/);
  assert.deepEqual(parseDiscount("100"), { set: true, basisPoints: 10000, issue: "" });
});

test("profile validation accepts valid phone and identifies invalid brand discounts", () => {
  const profile = normalizeOrderProfile({ contact_phone: "+7 (999) 123-45-67" });
  assert.equal(orderProfileIssue(profile, { angiopharm: "10", skin_synergy: "", klapp: "25.5" }), "");
  assert.match(orderProfileIssue(profile, { angiopharm: "", skin_synergy: "bad", klapp: "" }), /^Skin Synergy:/);
  assert.match(orderProfileIssue(normalizeOrderProfile({ contact_phone: "12345" }), {}), /Проверьте телефон/);
});

test("profileForSave serializes terms for all configured brands", () => {
  const saved = profileForSave(normalizeOrderProfile({ brand_terms: [{ brand: "klapp", dealer_name: "Dealer" }] }), {
    angiopharm: "25",
    skin_synergy: "1,5",
    klapp: "",
  });
  assert.deepEqual(saved.brand_terms.map(({ brand, discount_set, discount_basis_points }) => ({ brand, discount_set, discount_basis_points })), [
    { brand: "angiopharm", discount_set: true, discount_basis_points: 2500 },
    { brand: "skin_synergy", discount_set: true, discount_basis_points: 150 },
    { brand: "klapp", discount_set: false, discount_basis_points: 0 },
  ]);
  assert.equal(saved.brand_terms[2].dealer_name, "Dealer");
});
