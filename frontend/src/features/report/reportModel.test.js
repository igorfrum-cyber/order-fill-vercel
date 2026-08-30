import test from "node:test";
import assert from "node:assert/strict";

import { combinedSummary, jobProgress, jobStatusText, reportSummaryFromRows, statusLabel } from "./reportModel.js";

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

test("combinedSummary prefers the engine duplicate count over synthetic source rows", () => {
  const rows = [
    { status: "left_blank_nonpositive", duplicate: true },
    { status: "source_duplicate", duplicate: true },
  ];
  const summary = combinedSummary([
    { summary: { duplicates: 1, filled: 0, leftBlank: 1, unmatched: 0 } },
  ], rows);

  assert.equal(summary.duplicates, 1);
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

test("combinedSummary prefers the engine notInBlank count over sampled report rows", () => {
  const rows = [{ status: "not_in_blank" }, { status: "not_in_blank" }];
  const summary = combinedSummary([{ summary: { notInBlank: 12 }, reportRows: rows }], rows);
  assert.equal(summary.notInBlank, 12);
});

test("statusLabel falls back to raw status", () => {
  assert.equal(statusLabel("matched"), "Заполнено");
  assert.equal(statusLabel("custom"), "custom");
});

test("jobStatusText maps terminal errors to their API message", () => {
  assert.equal(jobStatusText({ status: "queued" }), "Задача в очереди...");
  assert.equal(jobStatusText({ status: "failed", error: { message: "Нет файла" } }), "Нет файла");
});

test("jobStatusText prefers the live progress message from the worker", () => {
  assert.equal(
    jobStatusText({ status: "processing", progress_message: "Читаю таблицу заказа" }),
    "Читаю таблицу заказа",
  );
});

test("jobProgress uses the worker-reported fraction instead of a fake status mapping", () => {
  assert.equal(jobProgress({ status: "processing", progress: 0.42 }), 0.42);
  assert.equal(jobProgress({ status: "processing", progress: 0.08 }), 0.08);
  assert.equal(jobProgress({ status: "processing", progress: 0 }), 0.12);
  assert.equal(jobProgress({ status: "queued" }), 0.05);
  assert.equal(jobProgress({ status: "processing" }), 0.12);
  assert.equal(jobProgress({ status: "needs_review" }), 1);
  assert.equal(jobProgress({ status: "needs_review", progress: 0.9 }), 1);
  assert.equal(jobProgress({ status: "completed" }), 1);
  assert.equal(jobProgress({ status: "failed" }), 1);
});
