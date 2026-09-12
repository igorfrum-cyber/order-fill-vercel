import test from "node:test";
import assert from "node:assert/strict";

import { defaultNorthActual, formatNorthQuantity, normalizeNorthResult, recalculateNorthRow } from "./northPlan.js";

test("formatNorthQuantity keeps integers compact and rounds decimals", () => {
  assert.equal(formatNorthQuantity(3), "3");
  assert.equal(formatNorthQuantity(3.125), "3.13");
  assert.equal(formatNorthQuantity(0), "");
});

test("recalculateNorthRow allocates free Tyumen stock before supplier need", () => {
  const row = {
    key: "a1",
    tyumenStock: 20,
    tyumenInTransit: 0,
    tyumenTarget: 5,
    supplierUnitSize: 1,
  };

  const result = recalculateNorthRow(row, { surgut: 10, urengoy: 10 });

  assert.equal(result.fromTyumen, 15);
  assert.equal(result.supplierNeed, 5);
  assert.deepEqual(result.supplierParts, [{ key: "surgut", label: "Сургут", quantity: 5 }]);
});

test("defaultNorthActual applies KLAPP nearest multiple rule", () => {
  assert.equal(defaultNorthActual({ supplierNeed: 10 }, 10, { kind: "klapp" }), 9);
});

test("defaultNorthActual matches origin/main north rounding", () => {
  assert.equal(defaultNorthActual({}, 4.75, { brand: "christina" }), 6);
  assert.equal(defaultNorthActual({ blankBoxSize: 6 }, 4.75, { brand: "angiopharm" }), 6);
  assert.equal(defaultNorthActual({ blankBoxSize: 4 }, 4.75, { brand: "levissime" }), 8);
  assert.equal(defaultNorthActual({}, 0.22, { brand: "skin_synergy" }), 1);
  assert.equal(defaultNorthActual({ novacutanMinimum: 100 }, 104, { kind: "novacutan" }), 100);
  assert.equal(defaultNorthActual({ novacutanMinimum: 10, supplierUnitSize: 5 }, 12, { brand: "novacutan" }), 12);
  assert.equal(defaultNorthActual({ blankBoxSize: 6 }, 2, { brand: "angiopharm", variant: "home" }), 3);
});

test("recalculateNorthRow caps free stock by warehouse location", () => {
  const result = recalculateNorthRow({
    key: "a1",
    tyumenStock: 20,
    tyumenInTransit: 0,
    tyumenTarget: 5,
    supplierUnitSize: 1,
    hasWarehouseStock: true,
    warehouseStock: 10,
    warehouseTransit: 0,
  }, { surgut: 10, urengoy: 10 });
  assert.equal(result.fromTyumen, 10);
  assert.equal(result.supplierNeed, 10);
});

test("normalizeNorthResult accepts snake_case API payload", () => {
  const result = normalizeNorthResult({
    has_tyumen_source: true,
    uploaded_cities: ["Тюмень"],
    plan_rows: [{ key: "a1" }],
    confirmation_groups: [{ city: { label: "Тюмень" }, variants: ["HOME"] }],
  }, { brand: "christina" });

  assert.equal(result.hasTyumenSource, true);
  assert.deepEqual(result.uploadedCities, ["Тюмень"]);
  assert.equal(result.planRows[0].key, "a1");
  assert.equal(result.summary.kind, "christina");
});
