import test from "node:test";
import assert from "node:assert/strict";

import { runOrderFillJob, loadOrderResume, resumeStage } from "./orderJobWorkflow.js";

test("runOrderFillJob creates a job, reports status updates, and returns normalized rows", async () => {
  const statuses = [];
  const api = {
    createOrderFillJob: async () => ({ id: "job-1" }),
    pollJob: async (jobId, { onUpdate }) => {
      assert.equal(jobId, "job-1");
      onUpdate({ status: "processing" });
      return { id: "job-1", status: "needs_review", brand: "angiopharm", order_month: "2026-08" };
    },
    getJobReport: async (jobId) => {
      assert.equal(jobId, "job-1");
      return { jobId, summary: {}, rows: [{ key: "row-1", status: "matched", blankRow: 4, editable: true }] };
    },
  };

  const result = await runOrderFillJob({
    api,
    command: { sourceFile: "source", blankFiles: ["blank"] },
    onStatus: (text) => statuses.push(text),
  });

  assert.deepEqual(statuses, ["Обработка..."]);
  assert.equal(result.jobId, "job-1");
  assert.equal(result.rows[0].blankRow, 4);
  assert.equal(result.results[0].summary.brand, "angiopharm");
});

test("runOrderFillJob throws failed job API message", async () => {
  const api = {
    createOrderFillJob: async () => ({ id: "job-1" }),
    pollJob: async () => ({ id: "job-1", status: "failed", error: { message: "Нет бланка" } }),
    getJobReport: async () => {
      throw new Error("must not load report");
    },
  };

  await assert.rejects(
    runOrderFillJob({
      api,
      command: { sourceFile: "source", blankFiles: ["blank"] },
    }),
    /Нет бланка/,
  );
});

test("loadOrderResume surfaces failed job error without loading a missing report", async () => {
  let reportCalls = 0;
  const loaded = await loadOrderResume("job-failed", {
    getJob: async () => ({
      id: "job-failed",
      status: "failed",
      brand: "christina",
      order_month: "2026-08",
      error: { message: "Не нашли колонку артикула в бланке." },
    }),
    getJobReport: async () => {
      reportCalls += 1;
      throw new Error("must not load report");
    },
    listJobFiles: async () => {
      throw new Error("must not list files");
    },
  });

  assert.equal(reportCalls, 0);
  assert.equal(loaded.status, "failed");
  assert.equal(loaded.error, "Не нашли колонку артикула в бланке.");
  assert.equal(loaded.finalized, false);
  assert.deepEqual(loaded.rows, []);
  assert.equal(resumeStage(loaded), "failed");
});

test("loadOrderResume does not treat needs_review without a report as an empty review", async () => {
  const loaded = await loadOrderResume("job-review", {
    getJob: async () => ({
      id: "job-review",
      type: "order_fill",
      status: "needs_review",
      brand: "",
      order_month: "",
    }),
    getJobReport: async () => {
      throw new Error("report was not found");
    },
    listJobFiles: async () => {
      throw new Error("must not list files");
    },
  });

  assert.equal(loaded.status, "needs_review");
  assert.match(loaded.error, /отчёт/i);
  assert.deepEqual(loaded.rows, []);
  assert.equal(loaded.finalized, false);
  assert.equal(resumeStage(loaded), "failed");
});

test("loadOrderResume still opens needs_review when the report exists", async () => {
  const loaded = await loadOrderResume("job-ok", {
    getJob: async () => ({
      id: "job-ok",
      status: "needs_review",
      brand: "christina",
      order_month: "2026-08",
    }),
    getJobReport: async () => ({
      summary: { brand: "christina" },
      rows: [{ key: "row-1" }],
    }),
    listJobFiles: async () => {
      throw new Error("must not list files");
    },
  });

  assert.equal(loaded.error, "");
  assert.equal(loaded.rows[0].key, "row-1");
  assert.equal(resumeStage(loaded), "fill");
});

test("loadOrderResume surfaces a failed north merge error without opening a blank review", async () => {
  let reportCalls = 0;
  const loaded = await loadOrderResume("job-north", {
    getJob: async () => ({
      id: "job-north",
      type: "north_merge",
      status: "failed",
      brand: "christina",
      error: { message: "не узнали город по имени файла \"Актуальный_бланк PROFF.xlsx\"." },
    }),
    getJobReport: async () => {
      reportCalls += 1;
      throw new Error("must not load report");
    },
    listJobFiles: async () => {
      throw new Error("must not list files");
    },
  });

  assert.equal(reportCalls, 0);
  assert.match(loaded.error, /не узнали город/);
  assert.equal(resumeStage(loaded), "failed");
});
