import test from "node:test";
import assert from "node:assert/strict";

import { runOrderFillJob } from "./orderJobWorkflow.js";

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
    command: { brand: "angiopharm", orderMonth: "2026-08", sourceFile: "source", blankFiles: ["blank"] },
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
      command: { brand: "angiopharm", orderMonth: "2026-08", sourceFile: "source", blankFiles: ["blank"] },
    }),
    /Нет бланка/,
  );
});
