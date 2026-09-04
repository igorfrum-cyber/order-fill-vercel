import test from "node:test";
import assert from "node:assert/strict";

import { combinedSummary, initialComment, jobProgress, jobStatusText, jobStatusLabel, jobsEmptyState, jobStatusHint, liveJobs, reportSummaryFromRows, statusLabel } from "./reportModel.js";

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

test("statusLabel never shows a raw API code", () => {
  assert.equal(statusLabel("matched"), "Заполнено");
  assert.equal(statusLabel("custom"), "Другое");
});

test("jobStatusLabel is Russian for every job status", () => {
  assert.equal(jobStatusLabel("queued"), "В очереди");
  assert.equal(jobStatusLabel("processing"), "Обработка");
  assert.equal(jobStatusLabel("needs_review"), "На проверке");
  assert.equal(jobStatusLabel("finalizing"), "Готовлю файлы");
  assert.equal(jobStatusLabel("completed"), "Готово");
  assert.equal(jobStatusLabel("failed"), "Ошибка");
  assert.equal(jobStatusLabel("mystery_code"), "В работе");
});

test("jobsEmptyState tells company users how to start and platform admin that the company has none", () => {
  assert.equal(
    jobsEmptyState("purchaser"),
    "Пока нет выгрузок. Начните с бланка закупки или объединения Севера.",
  );
  assert.equal(
    jobsEmptyState("company_admin"),
    "Пока нет выгрузок. Начните с бланка закупки или объединения Севера.",
  );
  assert.equal(jobsEmptyState("platform_admin"), "Пока нет выгрузок по выбранной компании.");
});

test("jobStatusHint explains each status without putting a paragraph in the row", () => {
  assert.equal(jobStatusHint("queued"), "Файлы ждут обработки.");
  assert.equal(jobStatusHint("processing"), "Сервис читает файлы и считает количества.");
  assert.equal(jobStatusHint("needs_review"), "Откройте выгрузку и проверьте строки.");
  assert.equal(jobStatusHint("finalizing"), "Собираю готовые Excel-файлы.");
  assert.equal(jobStatusHint("completed"), "Можно скачать готовые файлы.");
  assert.equal(jobStatusHint("failed"), "Не получилось обработать. Откройте строку, чтобы увидеть причину.");
  assert.equal(jobStatusHint("unknown"), "");
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

test("liveJobs keeps only work that is still in flight", () => {
  assert.deepEqual(
    liveJobs([
      { id: "1", status: "queued" },
      { id: "2", status: "completed" },
      { id: "3", status: "processing" },
      { id: "4", status: "failed" },
      { id: "5", status: "needs_review" },
    ]).map((job) => job.id),
    ["1", "3", "5"],
  );
});

test("initialComment prefers the 1C table comment over the auto box note", () => {
  assert.equal(initialComment({ sourceComment: "договорились", autoComment: "до коробки" }), "договорились");
  assert.equal(initialComment({ sourceComment: "", autoComment: "до коробки" }), "до коробки");
  assert.equal(initialComment({}), "");
});
