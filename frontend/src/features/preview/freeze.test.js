import test from "node:test";
import assert from "node:assert/strict";

import { buildRowOffsets, PREVIEW_HEADER_HEIGHT, PREVIEW_ROW_HEIGHT } from "./viewport.js";
import { freezeChromeHeight, isHeaderPinned } from "./freeze.js";

test("isHeaderPinned stays off until freeze is enabled", () => {
  const offsets = buildRowOffsets(20, 20);
  assert.equal(isHeaderPinned({ freeze: false, headerRow: 1, scrollTop: 40, offsets }), false);
});

test("isHeaderPinned ignores sheets without a title row", () => {
  const offsets = buildRowOffsets(20, 20);
  assert.equal(isHeaderPinned({ freeze: true, headerRow: 0, scrollTop: 80, offsets }), false);
});

test("isHeaderPinned sticks a first-row title immediately", () => {
  const offsets = buildRowOffsets(20, 28);
  assert.equal(isHeaderPinned({ freeze: true, headerRow: 1, scrollTop: 0, offsets }), true);
});

test("isHeaderPinned waits until the title row reaches the letter bar", () => {
  const offsets = buildRowOffsets(40, 20);
  assert.equal(isHeaderPinned({ freeze: true, headerRow: 14, scrollTop: 0, offsets }), false);
  assert.equal(isHeaderPinned({ freeze: true, headerRow: 14, scrollTop: 13 * 20 - 1, offsets }), false);
  assert.equal(isHeaderPinned({ freeze: true, headerRow: 14, scrollTop: 13 * 20, offsets }), true);
});

test("freezeChromeHeight keeps only the letter bar until the title sticks", () => {
  assert.equal(freezeChromeHeight({ pinned: false }), PREVIEW_HEADER_HEIGHT);
  assert.equal(
    freezeChromeHeight({ pinned: true, headerRowHeight: 36 }),
    PREVIEW_HEADER_HEIGHT + 36,
  );
  assert.equal(freezeChromeHeight({ pinned: true, headerRowHeight: PREVIEW_ROW_HEIGHT }), PREVIEW_HEADER_HEIGHT + PREVIEW_ROW_HEIGHT);
});
