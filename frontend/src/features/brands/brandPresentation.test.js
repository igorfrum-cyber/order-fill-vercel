import test from "node:test";
import assert from "node:assert/strict";

import { adjustmentLabelForBrand, mainBlankLabelForBrand, usesChristinaSplitBlank } from "./brandPresentation.js";

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
