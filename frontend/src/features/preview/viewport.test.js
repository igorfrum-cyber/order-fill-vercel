import test from "node:test";
import assert from "node:assert/strict";

import { columnName, columnLetters } from "./columns.js";
import { missingRange, visibleWindow } from "./viewport.js";
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
});
