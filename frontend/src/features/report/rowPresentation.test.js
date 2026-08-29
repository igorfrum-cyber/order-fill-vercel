import test from "node:test";
import assert from "node:assert/strict";

import {
  countByTab,
  presentationStatus,
  rowMatchesQuery,
  rowMatchesTab,
  visibleReportRows,
} from "./rowPresentation.js";

test("presentationStatus maps API row states onto fill-stage tabs", () => {
  assert.equal(presentationStatus({ status: "matched", inserted: 12 }), "filled");
  assert.equal(presentationStatus({ status: "matched_by_name", inserted: 4 }), "filled");
  assert.equal(presentationStatus({ status: "left_blank_nonpositive" }), "empty");
  assert.equal(presentationStatus({ status: "warning_name_differs" }), "check");
  assert.equal(presentationStatus({ status: "warning_name_only" }), "check");
  assert.equal(presentationStatus({ status: "not_in_source" }), "not_in_table");
  assert.equal(presentationStatus({ status: "not_in_blank" }), "not_in_blank");
  assert.equal(presentationStatus({ status: "matched", duplicate: true, inserted: 2 }), "duplicate");
  assert.equal(presentationStatus({ status: "source_duplicate" }), "duplicate");
});

test("countByTab and rowMatchesTab follow presentation status, not raw API status", () => {
  const rows = [
    { status: "matched", inserted: 12, blankName: "Крем", blankArticle: "A1" },
    { status: "left_blank_nonpositive", blankName: "Тоник", blankArticle: "A2" },
    { status: "warning_name_only", blankName: "Сыворотка", blankArticle: "A3" },
    { status: "not_in_source", blankName: "Маска", blankArticle: "A4" },
    { status: "not_in_blank", blankName: "Флюид", blankArticle: "B1" },
    { status: "matched", duplicate: true, inserted: 6, blankName: "Пилинг", blankArticle: "A5" },
  ];

  const counts = countByTab(rows);
  assert.equal(counts.all, 6);
  assert.equal(counts.filled, 1);
  assert.equal(counts.empty, 1);
  assert.equal(counts.check, 1);
  assert.equal(counts.duplicate, 1);
  assert.equal(counts.not_in_table, 1);
  assert.equal(counts.not_in_blank, 1);
  assert.equal(rows.filter((row) => rowMatchesTab(row, "empty")).length, 1);
});

test("visibleReportRows filters by tab and searches article or name", () => {
  const rows = [
    { status: "left_blank_nonpositive", blankName: "Крем для лица", blankArticle: "AP-100" },
    { status: "left_blank_nonpositive", blankName: "Тоник", blankArticle: "CS-200" },
    { status: "matched", inserted: 3, blankName: "Крем ночной", blankArticle: "AP-300" },
  ];

  assert.equal(visibleReportRows(rows, { tab: "empty", query: "крем" }).length, 1);
  assert.equal(visibleReportRows(rows, { tab: "all", query: "AP-" }).length, 2);
  assert.equal(rowMatchesQuery(rows[1], "тоник"), true);
});
