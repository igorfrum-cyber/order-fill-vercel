import test from "node:test";
import assert from "node:assert/strict";

import {
  createNorthMergeJob,
  createOrderFillJob,
  DEFAULT_POLL_INTERVAL_MS,
  DEFAULT_POLL_TIMEOUT_MS,
  downloadJobFile,
  FINALIZE_DONE_STATUSES,
  getJob,
  getJobReport,
  isPollDone,
  jobFileDownloadPath,
  listJobFiles,
  mapJobReportPayload,
  pollJob,
  submitJobEdits,
} from "./jobs.js";
import { apiClient } from "./client.js";

function stubClient(responses = [{}]) {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = async (url, options) => {
    calls.push({ url, options });
    const data = responses.shift() ?? {};
    return {
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => data,
      blob: async () => new Blob(["file"]),
    };
  };
  return {
    calls,
    restore() {
      apiClient.fetcher = originalFetcher;
      apiClient.baseUrl = originalBase;
    },
  };
}

test("jobFileDownloadPath points to a single generated file, not an archive", () => {
  assert.equal(jobFileDownloadPath("job 1", "output/2"), "/api/v1/jobs/job%201/files/output%2F2");
});

test("isPollDone treats needs_review as finished during the first fill", () => {
  assert.equal(isPollDone("needs_review"), true);
  assert.equal(isPollDone("completed"), true);
  assert.equal(isPollDone("failed"), true);
  assert.equal(isPollDone("processing"), false);
  assert.equal(isPollDone("finalizing"), false);
});

test("isPollDone waits past needs_review when finalizing reviewer edits", () => {
  const until = ["completed", "failed"];
  assert.equal(isPollDone("needs_review", until), false);
  assert.equal(isPollDone("finalizing", until), false);
  assert.equal(isPollDone("processing", until), false);
  assert.equal(isPollDone("completed", until), true);
  assert.equal(isPollDone("failed", until), true);
});

test("mapJobReportPayload preserves the North plan contract", () => {
  const payload = { has_tyumen_source: true, plan_rows: [{ key: "A1" }] };
  assert.equal(mapJobReportPayload(payload), payload);
});

test("job polling defaults and finalization statuses are stable", () => {
  assert.equal(DEFAULT_POLL_INTERVAL_MS, 500);
  assert.equal(DEFAULT_POLL_TIMEOUT_MS, 600000);
  assert.deepEqual(FINALIZE_DONE_STATUSES, ["completed", "failed"]);
});

test("createOrderFillJob uploads the source and all blanks with optional fields", async () => {
  const stub = stubClient();
  try {
    await createOrderFillJob({
      sourceFile: "sales.xlsx",
      blankFiles: ["blank-a.xlsx", "blank-b.xlsx"],
      warehouseFile: "warehouse.xlsx",
      companyId: "company-1",
    });
    const { url, options } = stub.calls[0];
    assert.equal(url, "/api/v1/jobs/order-fill");
    assert.equal(options.method, "POST");
    assert.deepEqual(options.body.getAll("blank_files"), ["blank-a.xlsx", "blank-b.xlsx"]);
    assert.equal(options.body.get("source_file"), "sales.xlsx");
    assert.equal(options.body.get("warehouse_file"), "warehouse.xlsx");
    assert.equal(options.body.get("company_id"), "company-1");
  } finally {
    stub.restore();
  }
});

test("createNorthMergeJob selects the matching blank fields and optional uploads", async () => {
  const stub = stubClient();
  try {
    await createNorthMergeJob({
      brand: "christina",
      blankFiles: [
        { variant: "home", file: "home.xlsx" },
        { variant: "proff", file: "proff.xlsx" },
        { variant: "other", file: "other.xlsx" },
      ],
      tyumenSourceFile: "tyumen.xlsx",
      warehouseFile: "warehouse.xlsx",
      companyId: "company-1",
    });
    const body = stub.calls[0].options.body;
    assert.equal(stub.calls[0].url, "/api/v1/jobs/north-merge");
    assert.equal(body.get("brand"), "christina");
    assert.equal(body.get("blank_home_files"), "home.xlsx");
    assert.equal(body.get("blank_proff_files"), "proff.xlsx");
    assert.equal(body.get("blank_files"), "other.xlsx");
    assert.equal(body.get("tyumen_source_file"), "tyumen.xlsx");
    assert.equal(body.get("warehouse_file"), "warehouse.xlsx");
    assert.equal(body.get("company_id"), "company-1");
  } finally {
    stub.restore();
  }
});

test("job endpoints encode identifiers and map report and file responses", async () => {
  const stub = stubClient([
    { id: "j/1", status: "processing" },
    { rows: [] },
    { files: [{ id: "f/1", name: "output.xlsx", download_path: "/download" }] },
  ]);
  try {
    assert.deepEqual(await getJob("j/1"), { id: "j/1", status: "processing" });
    const report = await getJobReport("j/1");
    assert.equal(report.jobId, "");
    assert.deepEqual(report.rows, []);
    assert.equal(report.summary.filled, 0);
    const files = await listJobFiles("j/1");
    assert.equal(files.files[0].id, "f/1");
    assert.equal(files.files[0].label, "output.xlsx");
    assert.equal(files.files[0].downloadUrl, "/download");
    assert.equal(stub.calls[0].url, "/api/v1/jobs/j%2F1");
    assert.equal(stub.calls[1].url, "/api/v1/jobs/j%2F1/report");
    assert.equal(stub.calls[2].url, "/api/v1/jobs/j%2F1/files");
  } finally {
    stub.restore();
  }
});

test("submitJobEdits serializes contract edit values and downloadJobFile delegates to download client", async () => {
  const stub = stubClient();
  try {
    await submitJobEdits("job/1", [{ key: "r1", value: 4 }]);
    const download = await downloadJobFile("job/1", "file/2");
    assert.equal(stub.calls[0].url, "/api/v1/jobs/job%2F1/edits");
    assert.equal(stub.calls[0].options.method, "POST");
    assert.deepEqual(JSON.parse(stub.calls[0].options.body), {
      edits: [{ key: "r1", value: "4", comment: "" }],
    });
    assert.equal(stub.calls[1].url, "/api/v1/jobs/job%2F1/files/file%2F2");
    assert.equal(download.blob.size, 4);
  } finally {
    stub.restore();
  }
});

test("pollJob reports changed progress and resolves on a terminal status", async () => {
  const stub = stubClient([
    { status: "processing", progress: 0.2, message: "Разбор" },
    { status: "processing", progress: 0.2, message: "Разбор" },
    { status: "completed", progress: 1, message: "Готово" },
  ]);
  const originalWindow = globalThis.window;
  globalThis.window = { setTimeout: (callback) => callback() };
  const updates = [];
  try {
    const result = await pollJob("job-1", {
      intervalMs: 0,
      onUpdate: (job) => updates.push(job),
    });
    assert.equal(result.status, "completed");
    assert.equal(stub.calls.length, 3);
    assert.deepEqual(updates.map((job) => job.status), ["processing", "completed"]);
  } finally {
    stub.restore();
    if (originalWindow === undefined) delete globalThis.window;
    else globalThis.window = originalWindow;
  }
});

test("pollJob rejects when the polling deadline is already expired", async () => {
  const stub = stubClient();
  try {
    await assert.rejects(
      pollJob("job-1", { timeoutMs: 0 }),
      /Задача выполняется слишком долго/,
    );
    assert.equal(stub.calls.length, 0);
  } finally {
    stub.restore();
  }
});
