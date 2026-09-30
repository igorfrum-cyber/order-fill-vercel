import test from "node:test";
import assert from "node:assert/strict";

import {
  defaultNorthActual,
  demandPiecesFromSupplierUnits,
  formatNorthCommentQuantity,
  formatNorthQuantity,
  nearestNorthMultiple,
  northPlanComment,
  northSupplierOrderText,
  northTransferParts,
  northTyumenFreeStock,
  normalizeNorthResult,
  recalculateNorthRow,
  supplierPartsForNorthActual,
  supplierUnitsFromPieces,
} from "./northPlan.js";

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

test("normalizers support camelCase payloads, defaults and invalid quantities", () => {
  assert.deepEqual(normalizeNorthResult({
    hasTyumenSource: true,
    uploadedCities: ["Сургут"],
    planRows: [{ key: "row-1" }],
    transfers: ["transfer"],
    confirmationGroups: ["group"],
    summary: { brand: "klapp" },
  }, {}).summary, { brand: "klapp" });
  assert.equal(normalizeNorthResult({}, {}).summary.kind, "");
  assert.equal(formatNorthQuantity("bad"), "");
  assert.equal(formatNorthQuantity(-1), "");
  assert.equal(formatNorthQuantity(Infinity), "");
  assert.equal(formatNorthCommentQuantity(2.7), "3");
  assert.equal(formatNorthCommentQuantity(-2), "");
  assert.equal(formatNorthCommentQuantity("bad"), "");
});

test("supplier unit conversions reject nonpositive demand and handle unit sizes", () => {
  assert.equal(supplierUnitsFromPieces("bad"), null);
  assert.equal(supplierUnitsFromPieces(-2), null);
  assert.equal(supplierUnitsFromPieces(2.345, 1), 2.35);
  assert.equal(supplierUnitsFromPieces(11, 5), 3);
  assert.equal(supplierUnitsFromPieces(11, "bad"), 11);
  assert.equal(demandPiecesFromSupplierUnits("bad", 5), 0);
  assert.equal(demandPiecesFromSupplierUnits(-1, 5), 0);
  assert.equal(demandPiecesFromSupplierUnits(3, 5), 15);
  assert.equal(demandPiecesFromSupplierUnits(3, "bad"), 3);
});

test("north transfer and supplier text preserve display order and box explanations", () => {
  assert.deepEqual(northTransferParts({ cities: [
    { key: "urengoy", quantity: 3 },
    { key: "surgut", quantity: 1 },
    { key: "unknown", quantity: 99 },
  ] }), [
    { key: "surgut", label: "Сургут", quantity: 1 },
    { key: "nizhnevartovsk", label: "Вартовск", quantity: 0 },
    { key: "urengoy", label: "Уренгой", quantity: 3 },
  ]);
  assert.equal(northSupplierOrderText({ supplierNeed: 8, supplierUnitSize: 5 }, 3), "3 коробок по 5");
  assert.equal(northSupplierOrderText({ supplierNeed: 3, supplierUnitSize: 1 }, 3), "3");
  assert.equal(northSupplierOrderText({ supplierNeed: 2 }, 5), "2 + 3 (до минимального) = 5");
  assert.equal(northSupplierOrderText({ supplierNeed: 8, supplierUnitSize: 5 }, 12), "8 + 4 (до минимального) = 12 коробок по 5");
});

test("supplier parts and plan comments describe supplier, transfer and local quantities", () => {
  const row = {
    supplierNeed: 5,
    actualSupplierOrder: 8,
    supplierUnitSize: 2,
    supplierParts: [
      { key: "tyumen", quantity: 10 },
      { key: "surgut", label: "Сургут", quantity: 3 },
    ],
    cities: [{ key: "urengoy", quantity: 2 }],
    northNeed: 4,
    budgetComment: "Согласовано",
  };
  assert.deepEqual(supplierPartsForNorthActual(row), [
    { key: "tyumen", label: "Тюмень", quantity: 13 },
    { key: "surgut", label: "Сургут", quantity: 3 },
  ]);
  assert.deepEqual(supplierPartsForNorthActual(row, 3), [
    { key: "tyumen", label: "Тюмень", quantity: 3 },
    { key: "surgut", label: "Сургут", quantity: 3 },
  ]);
  assert.equal(northPlanComment(row), [
    "Заказать у поставщика: 5 + 3 (до минимального) = 8 коробок по 2",
    "Отправить в Уренгой: 2",
    "Оставить в Тюмени: 13",
    "Согласовано",
  ].join("\n"));
  assert.equal(northPlanComment({ northNeed: 1 }), "Закрывается остатком Тюмени");
  assert.equal(northPlanComment({ northNeed: 0 }), "");
});

test("nearest multiple and Tyumen free stock cover invalid steps and caps", () => {
  assert.equal(nearestNorthMultiple("bad", 3), "");
  assert.equal(nearestNorthMultiple(-1, 3), "");
  assert.equal(nearestNorthMultiple(2.25, "bad"), 2.25);
  assert.equal(nearestNorthMultiple(3, 0), 3);
  assert.equal(nearestNorthMultiple(1, 3), 3);
  assert.equal(nearestNorthMultiple(10.5, 3), 12);
  assert.equal(northTyumenFreeStock(10, 2, 3, 7), 8);
  assert.equal(northTyumenFreeStock(1, 0, 0, 5), 0);
  assert.equal(northTyumenFreeStock(20, 0, 0, 5, 3, 1), 4);
  assert.equal(northTyumenFreeStock(20, 0, 0, 5, -1, 0), 0);
});

test("default North quantity selects Christina, Novacutan and box rounding rules", () => {
  assert.equal(defaultNorthActual({}, 0, {}), "");
  assert.equal(defaultNorthActual({}, 4, { variant: "proff" }), 6);
  assert.equal(defaultNorthActual({ novacutanMinimum: 100 }, 106, { brand: "novacutan" }), 110);
  assert.equal(defaultNorthActual({ blankBoxSize: 4 }, 5, { brand: "other" }), 8);
  assert.equal(defaultNorthActual({ blankBoxSize: "bad" }, 5, { brand: "other" }), 5);
});

test("recalculateNorthRow uses uploaded Tyumen plan and returns city and stock partitions", () => {
  const row = recalculateNorthRow({
    tyumenStock: 12,
    tyumenInTransit: 1,
    tyumenPlannedOrder: 99,
    tyumenTarget: 3,
    supplierUnitSize: 5,
  }, { tyumen: 2, nizhnevartovsk: 4, urengoy: 3, surgut: 1 });

  assert.equal(row.tyumenPlannedOrder, 99);
  assert.equal(row.tyumenFree, 12);
  assert.equal(row.supplierDemandNeed, 2);
  assert.equal(row.supplierNeed, 1);
  assert.deepEqual(row.cities.map(({ key }) => key), ["tyumen", "surgut", "nizhnevartovsk", "urengoy"]);
  assert.equal(row.northNeed, 8);
  assert.equal(row.fromTyumen, 8);
  assert.equal(row.supplierNorthNeed, 0);
});
