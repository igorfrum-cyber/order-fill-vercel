import test from "node:test";
import assert from "node:assert/strict";

import { previewBodyState, previewLoadingHint, previewLoadingTitle } from "./previewStatus.js";

test("previewBodyState stays on loading until the first grid window arrives", () => {
  assert.equal(previewBodyState({ fileId: "output-1" }), "loading");
  assert.equal(previewBodyState({ fileId: "output-1", meta: { sheets: [] } }), "empty");
  assert.equal(
    previewBodyState({
      fileId: "output-1",
      meta: { sheets: [{ index: 0 }] },
      sheet: { index: 0, max_row: 40 },
    }),
    "loading",
  );
  assert.equal(
    previewBodyState({
      fileId: "output-1",
      meta: { sheets: [{ index: 0 }] },
      sheet: { index: 0, max_row: 40 },
      gridReady: true,
    }),
    "ready",
  );
});

test("previewBodyState prefers an error over a blank sheet", () => {
  assert.equal(previewBodyState({ error: "Файл уже недоступен.", fileId: "output-1", gridReady: true }), "error");
});

test("preview loading copy is visible Russian, not a faint technical hint", () => {
  assert.match(previewLoadingTitle, /сетк/i);
  assert.match(previewLoadingHint, /подожд/i);
});
