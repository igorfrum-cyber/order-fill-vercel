import assert from "node:assert/strict";
import test from "node:test";

import {
  excelAcceptHint,
  northDuplicateFileMessage,
  northMissingCityBlankMessage,
  northSelectedCount,
  northUploadSteps,
  orderSelectedCount,
  orderUploadSteps,
  selectedFileCountLabel,
} from "./uploadCopy.js";

test("orderUploadSteps list the sales table before the supplier blank", () => {
  assert.deepEqual(
    orderUploadSteps().map((step) => `${step.n}. ${step.title}`),
    ["1. Таблица продаж из 1С", "2. Бланк поставщика"],
  );
});

test("northUploadSteps list city blanks before the optional Tyumen table", () => {
  assert.deepEqual(
    northUploadSteps().map((step) => `${step.n}. ${step.title}`),
    ["1. Бланки городов", "2. Таблица Тюмени, если нужно учесть остатки"],
  );
});

test("excelAcceptHint names the accepted format in plain language", () => {
  assert.equal(excelAcceptHint, "Подходят Excel-файлы.");
});

test("selectedFileCountLabel reports how many files are attached", () => {
  assert.equal(selectedFileCountLabel(0), "Файлы не выбраны.");
  assert.equal(selectedFileCountLabel(1), "Выбран 1 файл.");
  assert.equal(selectedFileCountLabel(3), "Выбрано файлов: 3.");
});

test("orderSelectedCount counts the sales table and each attached blank", () => {
  assert.equal(orderSelectedCount(null, {}), 0);
  assert.equal(orderSelectedCount({ name: "sales.xlsx" }, { main: { name: "blank.xlsx" } }), 2);
  assert.equal(orderSelectedCount({ name: "sales.xlsx" }, { home: { name: "home.xlsx" }, proff: null }), 2);
});

test("northSelectedCount counts city blanks and the Tyumen table", () => {
  assert.equal(northSelectedCount({ files: [{ name: "surgut.xlsx" }], tyumenFile: { name: "tyumen.xlsx" } }), 2);
  assert.equal(northSelectedCount({ homeFiles: [{ name: "a.xlsx" }], proffFiles: [{ name: "b.xlsx" }] }), 2);
});

test("north upload errors stay inline instead of as alerts", () => {
  assert.equal(northDuplicateFileMessage, "Все выбранные бланки уже добавлены.");
  assert.equal(northMissingCityBlankMessage, "Добавьте хотя бы один бланк города.");
});
