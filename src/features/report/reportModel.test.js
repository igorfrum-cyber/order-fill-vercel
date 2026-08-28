import test from "node:test";
import assert from "node:assert/strict";

import { combinedSummary, jobStatusText, normalizeReportRow, reportSummaryFromRows, statusLabel } from "./reportModel.js";

test("normalizeReportRow maps API snake_case fields to UI camelCase fields", () => {
  const row = normalizeReportRow({
    key: "row-1",
    status: "matched",
    blank_id: "home",
    blank_row: 7,
    blank_article: "A1",
    blank_name: "Cream",
    source_row: 12,
    source_article: "A1",
    auto_comment: "до коробки",
    duplicate_candidates: [{ source_row: 12 }],
  });

  assert.equal(row.blankId, "home");
  assert.equal(row.blankRow, 7);
  assert.equal(row.blankArticle, "A1");
  assert.equal(row.sourceRow, 12);
  assert.equal(row.autoComment, "до коробки");
  assert.deepEqual(row.duplicateCandidates, [{ source_row: 12 }]);
});

test("reportSummaryFromRows derives dashboard metrics from API report rows", () => {
  const rows = [
    { status: "matched", inserted: 3 },
    { status: "matched_by_name", inserted: null },
    { status: "warning_name_only" },
    { status: "left_blank_nonpositive" },
    { status: "not_in_source" },
  ];

  assert.deepEqual(reportSummaryFromRows(rows, { brand: "angiopharm", order_month: "2026-08" }), {
    brand: "angiopharm",
    orderMonthLabel: "2026-08",
    actualMainPeriod: "",
    actualPreviousPeriod: "",
    sourceCity: "",
    cityRule: "",
    deliveryWeeks: "",
    blankDuplicateArticles: 0,
    filled: 1,
    leftBlank: 1,
    suspicious: 1,
    unmatched: 1,
  });
});

test("combinedSummary aggregates result summaries and report-only counters", () => {
  const rows = [
    { status: "not_in_blank" },
    { status: "matched", duplicate: true },
  ];
  const summary = combinedSummary([
    { summary: { filled: 2, leftBlank: 1, suspicious: 0, unmatched: 1, blankDuplicateArticles: 1 }, reportRows: rows },
  ], rows);

  assert.equal(summary.notInBlank, 1);
  assert.equal(summary.duplicates, 1);
  assert.equal(summary.blankDuplicateArticles, 1);
});

test("statusLabel falls back to raw status", () => {
  assert.equal(statusLabel("matched"), "Заполнено");
  assert.equal(statusLabel("custom"), "custom");
});

test("jobStatusText maps terminal errors to their API message", () => {
  assert.equal(jobStatusText({ status: "queued" }), "Задача в очереди...");
  assert.equal(jobStatusText({ status: "failed", error: { message: "Нет файла" } }), "Нет файла");
});
