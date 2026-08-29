import test from "node:test";
import assert from "node:assert/strict";

import {
  adjustmentLabelForBrand,
  blankSlotsForBrand,
  brandLabel,
  mainBlankLabelForBrand,
  ORDER_BRANDS,
  usesChristinaSplitBlank,
} from "./brandPresentation.js";

test("adjustmentLabelForBrand returns UI labels for known brands", () => {
  assert.equal(adjustmentLabelForBrand("levissime"), "Кол-во в уп.");
  assert.equal(adjustmentLabelForBrand("novacutan"), "Мин. заказ");
  assert.equal(adjustmentLabelForBrand("angiopharm"), "Шт. в коробке");
});

test("mainBlankLabelForBrand returns report label for single-blank brands", () => {
  assert.equal(mainBlankLabelForBrand("sothys"), "SOTHYS");
  assert.equal(mainBlankLabelForBrand("unknown"), "ANGIO");
});

test("usesChristinaSplitBlank only enables HOME/PROFF split for Christina", () => {
  assert.equal(usesChristinaSplitBlank("christina"), true);
  assert.equal(usesChristinaSplitBlank("angiopharm"), false);
});

test("ORDER_BRANDS is the catalog the setup stage can render", () => {
  assert.deepEqual(ORDER_BRANDS.map((item) => item.id), [
    "angiopharm",
    "christina",
    "levissime",
    "sothys",
    "novacutan",
    "skin_synergy",
    "klapp",
  ]);
  assert.equal(brandLabel("sothys"), "SOTHYS");
});

test("blankSlotsForBrand splits Christina into HOME and PROFF", () => {
  assert.deepEqual(blankSlotsForBrand("angiopharm").map((slot) => slot.id), ["main"]);
  assert.deepEqual(blankSlotsForBrand("christina").map((slot) => slot.id), ["home", "proff"]);
});
