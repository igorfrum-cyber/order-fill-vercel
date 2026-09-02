import test from "node:test";
import assert from "node:assert/strict";

import { mapJob, mapOutputFile, mapReport, toManualEditPayload } from "./mappers.js";

const absoluteUrl = (path) => `http://api.test${path}`;

test("mapReport converts the API contract into the shape the UI renders", () => {
  const report = mapReport({
    job_id: "job-1",
    summary: { brand: "ANGIOPHARM", order_month_label: "сентябрь 2026", left_blank: 2, blank_duplicate_articles: 1 },
    rows: [
      {
        key: "blank-1:7",
        status: "matched",
        blank_id: "blank-1",
        blank_row: 7,
        blank_article: "A1",
        blank_name: "Крем",
        blank_quantity_col: 4,
        source_row: 12,
        auto_comment: "до коробки",
        inserted: 12,
        duplicate_candidates: [{ source_row: 12, source_article: "A1", in_transit: "3" }],
      },
    ],
  });

  assert.equal(report.jobId, "job-1");
  assert.equal(report.summary.orderMonthLabel, "сентябрь 2026");
  assert.equal(report.summary.leftBlank, 2);
  assert.equal(report.summary.blankDuplicateArticles, 1);

  const row = report.rows[0];
  assert.equal(row.blankId, "blank-1");
  assert.equal(row.blankRow, 7);
  assert.equal(row.blankQuantityCol, 4);
  assert.equal(row.sourceRow, 12);
  assert.equal(row.autoComment, "до коробки");
  assert.equal(row.inserted, 12);
  assert.equal(row.editable, true);
  assert.deepEqual(row.duplicateCandidates, [
    { sourceRow: 12, sourceArticle: "A1", sourceName: "", recommended: null, rounded: null, stock: "", inTransit: "3" },
  ]);
});

test("mapReport keeps source identity for rows that are missing from the blank", () => {
  const report = mapReport({
    job_id: "job-1",
    rows: [
      {
        key: "blank-1:missing:12",
        status: "not_in_blank",
        source_row: 12,
        source_article: "A400",
        source_name: "Сыворотка",
        recommended: 8,
        stock: "2",
        editable: false,
      },
    ],
  });

  const row = report.rows[0];
  assert.equal(row.status, "not_in_blank");
  assert.equal(row.blankArticle, "");
  assert.equal(row.blankName, "");
  assert.equal(row.sourceArticle, "A400");
  assert.equal(row.sourceName, "Сыворотка");
  assert.equal(row.recommended, 8);
});

test("mapReport tolerates an empty report", () => {
  const report = mapReport({});
  assert.deepEqual(report.rows, []);
  assert.equal(report.summary.brand, "");
});

test("toManualEditPayload sends every edit value as contract text", () => {
  assert.deepEqual(toManualEditPayload({ key: "blank-1:7", value: 12, comment: "меньше склада" }), {
    key: "blank-1:7",
    value: "12",
    comment: "меньше склада",
  });
  assert.deepEqual(toManualEditPayload({ key: "blank-1:8", value: null, comment: undefined }), {
    key: "blank-1:8",
    value: "",
    comment: "",
  });
});

test("mapOutputFile turns the API resource path into an absolute download URL", () => {
  const file = mapOutputFile(
    {
      id: "output-1",
      label: "Скачать заполненный бланк",
      name: "Бланк заполненный.xlsx",
      content_type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      size_bytes: 2048,
      download_path: "/api/v1/jobs/job-1/files/output-1",
    },
    absoluteUrl,
  );

  assert.equal(file.downloadUrl, "http://api.test/api/v1/jobs/job-1/files/output-1");
  assert.equal(file.label, "Скачать заполненный бланк");
  assert.equal(file.sizeBytes, 2048);
});

test("mapJob keeps live progress from the worker", () => {
  const job = mapJob(
    {
      id: "job-1",
      type: "order_fill",
      status: "processing",
      progress: 0.37,
      progress_message: "Читаю таблицу заказа",
      input_files: [],
      output_files: [],
    },
    absoluteUrl,
  );
  assert.equal(job.progress, 0.37);
  assert.equal(job.progressMessage, "Читаю таблицу заказа");
});
