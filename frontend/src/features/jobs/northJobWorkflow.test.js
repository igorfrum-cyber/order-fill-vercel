import test from "node:test";
import assert from "node:assert/strict";

import { runNorthMergeJob } from "./northJobWorkflow.js";

test("runNorthMergeJob creates north job and returns normalized plan", async () => {
  const api = {
    createNorthMergeJob: async () => ({ id: "north-1" }),
    pollJob: async () => ({ id: "north-1", status: "needs_review", brand: "christina" }),
    getJobReport: async () => ({ uploaded_cities: ["Сургут"], plan_rows: [{ key: "row-1" }] }),
  };

  const result = await runNorthMergeJob({
    api,
    command: { brand: "christina", blankFiles: [{ file: "blank" }], tyumenSourceFile: null },
  });

  assert.equal(result.jobId, "north-1");
  assert.deepEqual(result.plan.uploadedCities, ["Сургут"]);
  assert.equal(result.plan.summary.kind, "christina");
});

test("runNorthMergeJob forwards polling status updates to the caller", async () => {
  const updates = [];
  const api = {
    createNorthMergeJob: async () => ({ id: "north-2" }),
    pollJob: async (_id, options) => {
      const job = { id: "north-2", status: "processing", progress: 0.5, progressMessage: "Собираю план" };
      options.onUpdate(job);
      return { id: "north-2", status: "completed" };
    },
    getJobReport: async () => ({}),
  };
  await runNorthMergeJob({ api, command: {}, onStatus: (...args) => updates.push(args) });
  assert.equal(updates.length, 1);
  assert.equal(updates[0][0], "Собираю план");
  assert.equal(updates[0][1].progress, 0.5);
});

test("runNorthMergeJob surfaces the terminal failure message", async () => {
  const api = {
    createNorthMergeJob: async () => ({ id: "north-3" }),
    pollJob: async () => ({ id: "north-3", status: "failed", error: { message: "Файл повреждён" } }),
    getJobReport: async () => { throw new Error("report must not be requested after failure"); },
  };
  await assert.rejects(runNorthMergeJob({ api, command: {} }), /Файл повреждён/);
});
