import assert from "node:assert/strict";
import test from "node:test";

import {
  excelAcceptHint,
  fileMatchesAccept,
  northDuplicateFileMessage,
  northMissingCityBlankMessage,
  northSelectedCount,
  northUploadSteps,
  orderSelectedCount,
  orderUploadSteps,
  sameSelectedFile,
  selectedFileCountLabel,
} from "./uploadCopy.js";

test("orderUploadSteps list the sales table before the supplier blank", () => {
  assert.deepEqual(
    orderUploadSteps().map((step) => `${step.n}. ${step.title}`),
    ["1. Таблица продаж из 1С", "2. Бланк поставщика"],
  );
});

test("northUploadSteps list city blanks before the optional Tyumen locations", () => {
  assert.deepEqual(
    northUploadSteps().map((step) => `${step.n}. ${step.title}`),
    [
      "1. Бланки городов",
      "2. Таблица офиса Тюмени, если нужно учесть остатки",
      "3. Таблица склада доставки, если Тюмень ведётся в двух местах",
    ],
  );
});

test("excelAcceptHint names the accepted format in plain language", () => {
  assert.equal(excelAcceptHint, "Подходят .xlsx и .xlsm.");
});

test("fileMatchesAccept uses the same extensions as the file input", () => {
  const accept = ".xlsx,.xlsm";
  assert.equal(fileMatchesAccept({ name: "order.xlsx" }, accept), true);
  assert.equal(fileMatchesAccept({ name: "order.XLSX" }, accept), true);
  assert.equal(fileMatchesAccept({ name: "legacy.xls" }, accept), false);
  assert.equal(fileMatchesAccept({ name: "notes.pdf" }, accept), false);
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
  assert.equal(orderSelectedCount({ name: "sales.xlsx" }, { main: { name: "blank.xlsx" } }, { name: "warehouse.xlsx" }), 3);
});

test("sameSelectedFile catches the same workbook chosen for both Tyumen locations", () => {
  const office = { name: "Тюмень.xlsx", size: 42, lastModified: 10 };
  assert.equal(sameSelectedFile(office, { ...office }), true);
  assert.equal(sameSelectedFile(office, { ...office, size: 99 }), true);
  assert.equal(sameSelectedFile(office, { ...office, name: "Склад.xlsx" }), false);
});

test("northSelectedCount counts city blanks and both Tyumen locations", () => {
  assert.equal(northSelectedCount({ files: [{ name: "surgut.xlsx" }], tyumenFile: { name: "tyumen.xlsx" }, warehouseFile: { name: "warehouse.xlsx" } }), 3);
  assert.equal(northSelectedCount({ homeFiles: [{ name: "a.xlsx" }], proffFiles: [{ name: "b.xlsx" }] }), 2);
});

test("north upload errors stay inline instead of as alerts", () => {
  assert.equal(northDuplicateFileMessage, "Все выбранные бланки уже добавлены.");
  assert.equal(northMissingCityBlankMessage, "Добавьте хотя бы один бланк города.");
});
