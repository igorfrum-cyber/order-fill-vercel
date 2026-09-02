import test from "node:test";
import assert from "node:assert/strict";

import { cellCss, mergeLayout, visibleMerges } from "./appearance.js";

test("cellCss maps interned appearance onto CSS", () => {
  const css = cellCss(
    [{}, { fill: "#1f4e79", color: "#ffffff", bold: true, align: "center", wrap: true, border_t: true }],
    1,
  );
  assert.equal(css.backgroundColor, "#1f4e79");
  assert.equal(css.color, "#ffffff");
  assert.equal(css.fontWeight, 700);
  assert.equal(css.justifyContent, "center");
  assert.equal(css.whiteSpace, "pre-wrap");
  assert.equal(css.borderTop, "1px solid #b4b4b4");
});

test("mergeLayout marks covered cells and keeps the origin", () => {
  const { covered, origins, list } = mergeLayout([{ row: 1, column: 1, height: 1, width: 2 }]);
  assert.equal(origins.has("1:1"), true);
  assert.equal(covered.has("1:1"), false);
  assert.equal(covered.has("1:2"), true);
  assert.equal(list.length, 1);
});

test("visibleMerges keeps spans that overlap the viewport", () => {
  const merges = [
    { row: 1, column: 1, height: 2, width: 2 },
    { row: 10, column: 1, height: 1, width: 3 },
  ];
  assert.equal(visibleMerges(merges, 2, 4).length, 1);
  assert.equal(visibleMerges(merges, 2, 4)[0].row, 1);
});
