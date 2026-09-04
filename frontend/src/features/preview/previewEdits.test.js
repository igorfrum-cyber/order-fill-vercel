import test from "node:test";
import assert from "node:assert/strict";

import { blankIdForPreviewFile, defaultPreviewFileId, findEditColumns, isCommentOverlay, isQuantityOverlay, isSourcePreviewFile, needsEditResubmit, orderSheetIndex, previewOverlays } from "./previewEdits.js";

const files = [
  { id: "output-1", label: "Скачать заполненный бланк", name: "бланк заполненный.xlsx" },
  { id: "output-2", label: "Скачать заполненную таблицу заказа", name: "Ангио заполненная таблица.xlsx" },
];

test("isSourcePreviewFile recognises the 1C table output", () => {
  assert.equal(isSourcePreviewFile(files[0]), false);
  assert.equal(isSourcePreviewFile(files[1]), true);
});

test("blankIdForPreviewFile maps the first blank file onto blank-1", () => {
  assert.equal(blankIdForPreviewFile(files, "output-1"), "blank-1");
  assert.equal(blankIdForPreviewFile(files, "output-2"), "");
});

test("previewOverlays mirrors fact onto the blank quantity cell without making the blank editable", () => {
  const rows = [
    {
      key: "blank-1:22",
      blankId: "blank-1",
      blankRow: 22,
      blankQuantityCol: 5,
      sourceRow: 15,
      editable: true,
      inserted: 12,
    },
  ];
  const edits = new Map([["blank-1:22", { value: 18, comment: "договорились" }]]);
  const blank = previewOverlays(rows, edits, { files, fileId: "output-1" }).get("22:5");
  const fact = previewOverlays(rows, edits, {
    files,
    fileId: "output-2",
    quantityColumn: 3,
    commentColumn: 4,
  }).get("15:3");
  assert.equal(fact.value, "18");
  assert.equal(isQuantityOverlay(fact), true);
  assert.equal(blank.value, "18");
  assert.equal(isQuantityOverlay(blank), false);
});

test("defaultPreviewFileId opens the 1C table so fact can be edited first", () => {
  assert.equal(defaultPreviewFileId(files), "output-2");
});

test("orderSheetIndex opens the sheet that has Заказано по факту", () => {
  assert.equal(
    orderSheetIndex([
      { index: 0, name: "Cover", header_row: 0 },
      { index: 1, name: "Тюмень", header_row: 8, quantity_column: 34 },
    ]),
    1,
  );
});

test("findEditColumns picks fact and comment and ignores recommended order", () => {
  const header = ["Артикул", "Рекомендованный заказ", "Заказано по факту", "Комментарий"];
  assert.deepEqual(findEditColumns(header), { quantity: 3, comment: 4 });
});

test("previewOverlays stays off unmatched blank cells", () => {
  const rows = [
    { key: "blank-1:22", blankId: "blank-1", blankRow: 22, blankQuantityCol: 5, editable: true, inserted: 12 },
  ];
  const edits = new Map([["blank-1:22", { value: 12, comment: "до коробки" }]]);
  assert.equal(previewOverlays(rows, edits, { files, fileId: "output-1" }).has("22:4"), false);
});

test("previewOverlays on the 1C table edits fact and comment, not the recommended column", () => {
  const rows = [
    {
      key: "blank-1:22",
      blankId: "blank-1",
      blankRow: 22,
      blankQuantityCol: 5,
      sourceRow: 15,
      editable: true,
      inserted: 12,
    },
  ];
  const edits = new Map([["blank-1:22", { value: 18, comment: "договорились" }]]);
  const overlays = previewOverlays(rows, edits, {
    files,
    fileId: "output-2",
    quantityColumn: 3,
    commentColumn: 4,
  });
  assert.equal(overlays.get("15:3").field, "value");
  assert.equal(overlays.get("15:3").value, "18");
  assert.equal(overlays.get("15:4").field, "comment");
  assert.equal(overlays.get("15:4").value, "договорились");
  assert.equal(overlays.has("15:2"), false);
});

test("isQuantityOverlay is only true for quantity cells", () => {
  assert.equal(isQuantityOverlay({ field: "value", key: "blank-1:22", value: "18" }), true);
  assert.equal(isCommentOverlay({ field: "comment", key: "blank-1:22", value: "ок" }), true);
  assert.equal(isQuantityOverlay({ field: "formula", value: "35" }), false);
  assert.equal(isQuantityOverlay({ field: "comment", key: "blank-1:22", value: "ок" }), false);
  assert.equal(isQuantityOverlay({ field: "value", value: "18" }), false);
});

test("needsEditResubmit is true for the first save and for dirty preview edits", () => {
  assert.equal(needsEditResubmit({ finalized: false, dirty: true, hasDeviations: true }), true);
  assert.equal(needsEditResubmit({ finalized: true, dirty: true, hasDeviations: true }), true);
  assert.equal(needsEditResubmit({ finalized: true, dirty: false, hasDeviations: true }), false);
  assert.equal(needsEditResubmit({ finalized: false, dirty: true, hasDeviations: false }), false);
});
