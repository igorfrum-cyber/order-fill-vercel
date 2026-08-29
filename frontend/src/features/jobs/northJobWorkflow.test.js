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
