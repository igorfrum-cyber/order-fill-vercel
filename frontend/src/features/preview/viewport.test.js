import test from "node:test";
import assert from "node:assert/strict";

import { columnName, columnLetters } from "./columns.js";
import { missingRange, visibleWindow, buildRowOffsets, parseCustomHeights, gridContentWidth, columnSize, columnOffsets, spanSize, PREVIEW_GUTTER_WIDTH } from "./viewport.js";
import { previewFileTitle } from "./fileTitle.js";

test("columnName matches spreadsheet letters through AI", () => {
  assert.equal(columnName(1), "A");
  assert.equal(columnName(26), "Z");
  assert.equal(columnName(27), "AA");
  assert.equal(columnName(34), "AH");
  assert.equal(columnName(35), "AI");
});

test("columnLetters lists every column up to the used range", () => {
  assert.deepEqual(columnLetters(3), ["A", "B", "C"]);
});

test("visibleWindow virtualizes a 100k sheet without enumerating rows", () => {
  const window = visibleWindow({
    scrollTop: 28 * 1000,
    viewportHeight: 28 * 20,
    rowHeight: 28,
    headerHeight: 28,
    maxRow: 100_000,
    buffer: 10,
  });
  assert.equal(window.fromRow, 990);
  assert.equal(window.toRow, 1029);
  assert.ok(window.toRow - window.fromRow < 80);
});

test("visibleWindow uses prefix offsets for mixed row heights", () => {
  const offsets = buildRowOffsets(20, 20, new Map([[1, 40], [2, 40]]));
  const window = visibleWindow({
    scrollTop: 28 + 80,
    viewportHeight: 60,
    headerHeight: 28,
    maxRow: 20,
    buffer: 0,
    offsets,
  });
  assert.equal(window.fromRow, 3);
  assert.ok(window.toRow >= 3);
});

test("parseCustomHeights reads JSON string keys", () => {
  const heights = parseCustomHeights({ 1: 40, "5": 32 });
  assert.equal(heights.get(1), 40);
  assert.equal(heights.get(5), 32);
});

test("gridContentWidth sums Excel column widths", () => {
  assert.equal(gridContentWidth(3, [135, 91, 91], 52), 52 + 135 + 91 + 91);
});

test("columnSize keeps narrow spacer columns instead of the 92px fallback", () => {
  assert.equal(columnSize(0, [10, 91]), 10);
  assert.equal(columnSize(1, [10, 0]), 0);
  assert.equal(columnSize(0, undefined), 92);
});

test("spanSize of a merge equals the union of its columns", () => {
  const offsets = columnOffsets(4, [79, 34, 426, 75]);
  assert.equal(spanSize(offsets, 0, 3), 79 + 34 + 426);
  assert.equal(PREVIEW_GUTTER_WIDTH + offsets[3], PREVIEW_GUTTER_WIDTH + 79 + 34 + 426);
});

test("missingRange skips rows already in the cache", () => {
  const cache = new Map([
    [10, ["A"]],
    [11, ["B"]],
    [12, ["C"]],
  ]);
  assert.equal(missingRange(cache, 10, 12), null);
  assert.deepEqual(missingRange(cache, 11, 20), { fromRow: 13, toRow: 20 });
});

test("previewFileTitle prefers a short tab label over the download CTA", () => {
  assert.equal(previewFileTitle({ label: "Скачать заполненный бланк", name: "Бланк.xlsx" }), "Бланк");
  assert.equal(previewFileTitle({ label: "Скачать заполненную таблицу заказа", name: "Заказ.xlsx" }), "Таблица 1С");
  assert.equal(previewFileTitle({ label: "Скачать файл", name: "Ангио заполненная таблица.xlsx" }), "Таблица 1С");
});
