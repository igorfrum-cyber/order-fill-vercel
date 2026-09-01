import assert from "node:assert/strict";
import test from "node:test";

import { parseArgs, percentile, summarizeResults } from "./load-order-fill.mjs";

test("parseArgs applies conservative defaults for a local load run", () => {
  const options = parseArgs([
    "--source",
    "testdata/private/source_100000.xlsx",
    "--blank",
    "testdata/private/blank_100000.xlsx",
  ]);

  assert.equal(options.apiBaseUrl, "http://127.0.0.1:8080");
  assert.equal(options.jobs, 20);
  assert.equal(options.concurrency, 5);
  assert.equal(options.brand, "angiopharm");
  const now = new Date();
  const next = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1));
  const expectedMonth = `${next.getUTCFullYear()}-${String(next.getUTCMonth() + 1).padStart(2, "0")}`;
  assert.equal(options.orderMonth, expectedMonth);
  assert.equal(options.sourcePath, "testdata/private/source_100000.xlsx");
  assert.deepEqual(options.blankPaths, ["testdata/private/blank_100000.xlsx"]);
  assert.equal(options.pollIntervalMs, 1000);
  assert.equal(options.timeoutMs, 10 * 60 * 1000);
  assert.equal(options.fetchReport, true);
  assert.equal(options.fetchFiles, true);
  assert.equal(options.downloadArchive, false);
});

test("parseArgs supports a 100-job burst with multiple blanks", () => {
  const options = parseArgs([
    "--api",
    "http://api.test",
    "--jobs",
    "100",
    "--concurrency",
    "100",
    "--brand",
    "christina",
    "--order-month",
    "2026-10",
    "--source",
    "source.xlsx",
    "--blank",
    "home.xlsx",
    "--blank",
    "proff.xlsx",
    "--poll-ms",
    "250",
    "--timeout-ms",
    "120000",
    "--download-archive",
    "--no-report",
    "--no-files",
  ]);

  assert.equal(options.apiBaseUrl, "http://api.test");
  assert.equal(options.jobs, 100);
  assert.equal(options.concurrency, 100);
  assert.equal(options.brand, "christina");
  assert.equal(options.orderMonth, "2026-10");
  assert.deepEqual(options.blankPaths, ["home.xlsx", "proff.xlsx"]);
  assert.equal(options.pollIntervalMs, 250);
  assert.equal(options.timeoutMs, 120000);
  assert.equal(options.downloadArchive, true);
  assert.equal(options.fetchReport, false);
  assert.equal(options.fetchFiles, false);
});

test("parseArgs rejects unsafe numeric arguments", () => {
  assert.throws(() => parseArgs(["--source", "s.xlsx", "--blank", "b.xlsx", "--jobs", "0"]), /--jobs/);
  assert.throws(() => parseArgs(["--source", "s.xlsx", "--blank", "b.xlsx", "--concurrency", "-1"]), /--concurrency/);
  assert.throws(() => parseArgs(["--source", "s.xlsx", "--blank", "b.xlsx", "--poll-ms", "0"]), /--poll-ms/);
});

test("percentile returns nearest-rank values", () => {
  const values = [100, 300, 200, 400, 500];

  assert.equal(percentile(values, 0.5), 300);
  assert.equal(percentile(values, 0.95), 500);
  assert.equal(percentile([], 0.95), 0);
});

test("summarizeResults separates accepted, completed and failed jobs", () => {
  const summary = summarizeResults([
    { ok: true, enqueueMs: 100, totalMs: 1000, status: "needs_review", reportRows: 7, files: 2 },
    { ok: true, enqueueMs: 300, totalMs: 2000, status: "completed", reportRows: 8, files: 2 },
    { ok: false, enqueueMs: 50, totalMs: 500, status: "failed", error: "bad workbook" },
  ]);

  assert.equal(summary.total, 3);
  assert.equal(summary.ok, 2);
  assert.equal(summary.failed, 1);
  assert.equal(summary.statuses.needs_review, 1);
  assert.equal(summary.statuses.completed, 1);
  assert.equal(summary.statuses.failed, 1);
  assert.equal(summary.enqueue.p50, 100);
  assert.equal(summary.enqueue.p95, 300);
  assert.equal(summary.completion.p50, 1000);
  assert.equal(summary.completion.p95, 2000);
  assert.equal(summary.reportRows, 15);
  assert.equal(summary.files, 4);
});
