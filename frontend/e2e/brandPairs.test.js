import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";

import {
  CHRISTINA_PROFF_UI_BLANK,
  CHRISTINA_PROFF_UI_SOURCE,
  brandKeyFromName,
  christinaProffUiPair,
  isChristinaProffBlank,
  matchBrandPairs,
  norm,
  resolvePrivateTestdata,
  sameWorkbookName,
  scanPrivateBrandPairs,
} from "./brandPairs.js";

test("norm is NFC, lowercases, maps ё, and drops punctuation", () => {
  const nfd = "Актуальный_бланк PROFF.xlsx".normalize("NFD");
  assert.equal(norm(nfd), "актуальный бланк proff xlsx");
  assert.equal(norm("Тюмень .xlsx"), "тюмень xlsx");
  assert.equal(norm("Крёстина"), "крестина");
});

test("Christina PROFF blank without the word Christina maps to christina", () => {
  assert.equal(brandKeyFromName("Актуальный_бланк PROFF.xlsx"), "christina");
  assert.equal(brandKeyFromName("Актуальный_бланк PROFF (1).xlsx"), "christina");
  assert.equal(brandKeyFromName("Актуальный_бланк PROFF.xlsx".normalize("NFD")), "christina");
  assert.equal(brandKeyFromName("бланк проф.xlsx"), "christina");
  assert.equal(brandKeyFromName("blank-prof.xlsx"), "christina");
  assert.equal(isChristinaProffBlank("Актуальный_бланк PROFF.xlsx"), true);
  assert.equal(brandKeyFromName("Кристина Сургут .xlsx"), "christina");
});

test("KLAPP Surgut and Tyumen share a brand key, city is not a key", () => {
  assert.equal(brandKeyFromName("Бланк Заказа KLAPP август 2026 (1).xlsx"), "klapp");
  assert.equal(brandKeyFromName("Клапп Тюмень .xlsx"), "klapp");
  assert.equal(brandKeyFromName("Сургут Клапп.xlsx"), "klapp");
  const pairs = matchBrandPairs(
    ["Бланк Заказа KLAPP август 2026 (1).xlsx"],
    ["Клапп Тюмень .xlsx", "Сургут Клапп.xlsx"],
  );
  assert.equal(pairs.length, 2);
  assert.deepEqual(pairs.map((pair) => pair.source).sort(), ["Клапп Тюмень .xlsx", "Сургут Клапп.xlsx"]);
  assert.ok(pairs.every((pair) => pair.brand === "klapp"));
});

test("does not cartesian angiopharm with klapp", () => {
  const pairs = matchBrandPairs(
    ["2026 08 25 Бланк заказа ANGIOPHARM.xlsx", "Бланк Заказа KLAPP август 2026 (1).xlsx"],
    ["Ангио Сургут.xlsx", "Клапп Тюмень .xlsx", "Сургут Клапп.xlsx"],
  );
  assert.equal(pairs.length, 3);
  assert.equal(pairs.filter((pair) => pair.brand === "angiopharm").length, 1);
  assert.equal(pairs.filter((pair) => pair.brand === "klapp").length, 2);
  assert.equal(
    pairs.some((pair) => pair.blank.includes("ANGIOPHARM") && /клапп|klapp/i.test(pair.source)),
    false,
  );
});

test("Christina two PROFF blanks × two sales tables is 4 pairs, not a 5×8 cartesian", () => {
  const pairs = matchBrandPairs(
    [
      "2026 08 25 Бланк заказа ANGIOPHARM.xlsx",
      "_Бланк заказа Skin Synergy от 26.08.2026.xlsx",
      "Актуальный_бланк PROFF.xlsx",
      "Актуальный_бланк PROFF (1).xlsx",
      "Бланк Заказа KLAPP август 2026 (1).xlsx",
    ],
    [
      "Ангио Сургут.xlsx",
      "Ангио Тюмень .xlsx",
      "Клапп Тюмень .xlsx",
      "Сургут Клапп.xlsx",
      "Кристина Сургут .xlsx",
      "Кристина Тюмень .xlsx",
      "Скин Синерджи Сургут.xlsx",
      "Скин Синерджи Тюмень .xlsx",
    ],
  );
  assert.equal(pairs.length, 10);
  assert.equal(pairs.filter((pair) => pair.brand === "christina").length, 4);
  assert.equal(pairs.filter((pair) => pair.brand === "angiopharm").length, 2);
  assert.equal(pairs.filter((pair) => pair.brand === "klapp").length, 2);
  assert.equal(pairs.filter((pair) => pair.brand === "skin_synergy").length, 2);
});

test("orphan names and mixed-brand names are not paired", () => {
  assert.equal(brandKeyFromName("source_1000.xlsx"), "");
  assert.equal(brandKeyFromName("ANGIOPHARM KLAPP.xlsx"), "");
  assert.deepEqual(matchBrandPairs(["без бренда.xlsx"], ["Ангио Тюмень .xlsx"]), []);
});

test("scanPrivateBrandPairs skips when the private folders are missing", () => {
  const scanned = scanPrivateBrandPairs("/tmp/order-fill-no-private-testdata");
  assert.equal(scanned.available, false);
  assert.deepEqual(scanned.pairs, []);
});

test("sameWorkbookName ignores Unicode NFD and trailing spaces", () => {
  assert.equal(sameWorkbookName(CHRISTINA_PROFF_UI_BLANK.normalize("NFD"), CHRISTINA_PROFF_UI_BLANK), true);
  assert.equal(sameWorkbookName(CHRISTINA_PROFF_UI_SOURCE, "Кристина Тюмень.xlsx"), true);
  assert.equal(sameWorkbookName(CHRISTINA_PROFF_UI_BLANK, "Актуальный_бланк PROFF.xlsx"), false);
});

test("christinaProffUiPair picks PROFF (1) × Кристина Тюмень", () => {
  const root = fs.mkdtempSync(path.join("/tmp", "order-fill-christina-ui-"));
  const blanks = path.join(root, "Бланки");
  const sales = path.join(root, "таблицы продаж");
  fs.mkdirSync(blanks);
  fs.mkdirSync(sales);
  fs.writeFileSync(path.join(blanks, CHRISTINA_PROFF_UI_BLANK.normalize("NFD")), "");
  fs.writeFileSync(path.join(blanks, "Актуальный_бланк PROFF.xlsx"), "");
  fs.writeFileSync(path.join(sales, CHRISTINA_PROFF_UI_SOURCE), "");
  fs.writeFileSync(path.join(sales, "Кристина Сургут .xlsx"), "");
  try {
    const { available, pair } = christinaProffUiPair(root);
    assert.equal(available, true);
    assert.ok(pair);
    assert.equal(pair.brand, "christina");
    assert.equal(sameWorkbookName(pair.blank, CHRISTINA_PROFF_UI_BLANK), true);
    assert.equal(sameWorkbookName(pair.source, CHRISTINA_PROFF_UI_SOURCE), true);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("resolvePrivateTestdata prefers ORDER_FILL_PRIVATE_TESTDATA", () => {
  assert.equal(
    resolvePrivateTestdata("/tmp/repo", { ORDER_FILL_PRIVATE_TESTDATA: "/custom/private" }),
    "/custom/private",
  );
});
