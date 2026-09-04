import test from "node:test";
import assert from "node:assert/strict";

import { formulaOverlays } from "./formulas.js";

test("formulaOverlays recomputes a column SUM after a quantity overlay", () => {
  const formulas = [{ row: 4, column: 5, formula: "SUM(E2:E3)" }];
  const values = { "2:5": "12", "3:5": "8" };
  const overlays = new Map([["2:5", { field: "value", value: "18" }]]);
  const derived = formulaOverlays(formulas, { overlays, values });
  assert.equal(derived.get("4:5").value, "26");
  assert.equal(derived.get("4:5").field, "formula");
});

test("formulaOverlays updates qty * price on the same row", () => {
  const formulas = [{ row: 2, column: 7, formula: "E2*F2" }];
  const values = { "2:5": "12", "2:6": "100" };
  const overlays = new Map([["2:5", { field: "value", value: "18" }]]);
  const derived = formulaOverlays(formulas, { overlays, values });
  assert.equal(derived.get("2:7").value, "1800");
});

test("formulaOverlays follows a chain: amount then total", () => {
  const formulas = [
    { row: 2, column: 7, formula: "E2*F2" },
    { row: 4, column: 7, formula: "SUM(G2:G3)" },
  ];
  const values = { "2:5": "12", "2:6": "100", "3:7": "200" };
  const overlays = new Map([["2:5", { field: "value", value: "18" }]]);
  const derived = formulaOverlays(formulas, { overlays, values });
  assert.equal(derived.get("2:7").value, "1800");
  assert.equal(derived.get("4:7").value, "2000");
});

test("formulaOverlays leaves a quantity edit cell alone", () => {
  const formulas = [{ row: 2, column: 5, formula: "SUM(E3:E4)" }];
  const overlays = new Map([["2:5", { field: "value", value: "18", key: "blank-1:2" }]]);
  const derived = formulaOverlays(formulas, { overlays, values: { "3:5": "1", "4:5": "1" } });
  assert.equal(derived.has("2:5"), false);
});

test("formulaOverlays recomputes nested SUBTOTAL rows without double-counting", () => {
  const formulas = [
    { row: 4, column: 5, formula: "SUBTOTAL(9,E2:E3)" },
    { row: 7, column: 5, formula: "SUBTOTAL(9,E5:E6)" },
    { row: 8, column: 5, formula: "SUBTOTAL(9,E2:E7)" },
  ];
  const values = { "2:5": "10", "3:5": "5", "5:5": "7", "6:5": "3" };
  const overlays = new Map([["2:5", { field: "value", value: "20" }]]);
  const derived = formulaOverlays(formulas, { overlays, values });
  assert.equal(derived.get("4:5").value, "25");
  assert.equal(derived.get("7:5").value, "10");
  assert.equal(derived.get("8:5").value, "35");
});

test("formulaOverlays sums a list of subtotal cells", () => {
  const formulas = [
    { row: 4, column: 5, formula: "SUBTOTAL(9,E2:E3)" },
    { row: 8, column: 5, formula: "E4+E7" },
  ];
  const values = { "2:5": "10", "3:5": "5", "7:5": "8" };
  const derived = formulaOverlays(formulas, { overlays: new Map(), values });
  assert.equal(derived.get("8:5").value, "23");
});

test("formulaOverlays skips a million-cell range instead of freezing the preview", () => {
  const started = Date.now();
  const derived = formulaOverlays([{ row: 1048577, column: 5, formula: "SUM(E1:E1048576)" }], {
    values: { "1:5": "1" },
  });
  assert.ok(Date.now() - started < 50);
  assert.equal(derived.has("1048577:5"), false);
});

test("formulaOverlays ignores a non-array formulas payload", () => {
  assert.equal(formulaOverlays({ 0: { row: 1, column: 1, formula: "A1" } }).size, 0);
});
