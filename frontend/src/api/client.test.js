import test from "node:test";
import assert from "node:assert/strict";

import { ApiClient, onAuthRequired } from "./client.js";

test("ApiClient invokes fetch with the global receiver expected by browser fetch", async () => {
  const fetcher = function fetcher() {
    assert.equal(this, globalThis);
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ status: "ok" }),
    });
  };

  const client = new ApiClient({ baseUrl: "http://api.example", fetcher });

  assert.deepEqual(await client.request("/healthz"), { status: "ok" });
});

test("ApiClient.requestDownload returns the blob and RFC 5987 file name", async () => {
  const blob = new Blob(["zip"]);
  const headers = {
    get(name) {
      if (name === "Content-Type") return "application/zip";
      if (name === "Content-Disposition") {
        return `attachment; filename="angio.zip"; filename*=UTF-8''angiopharm_2026-09.zip`;
      }
      return null;
    },
  };
  const fetcher = function fetcher(url) {
    assert.equal(this, globalThis);
    assert.equal(url, "http://api.example/api/v1/jobs/job-1/archive");
    return Promise.resolve({
      ok: true,
      status: 200,
      headers,
      blob: async () => blob,
    });
  };

  const client = new ApiClient({ baseUrl: "http://api.example", fetcher });
  const download = await client.requestDownload("/api/v1/jobs/job-1/archive");

  assert.equal(download.fileName, "angiopharm_2026-09.zip");
  assert.equal(download.contentType, "application/zip");
  assert.equal(download.blob, blob);
});

test("ApiClient.request notifies on expired session", async () => {
  const client = new ApiClient({
    baseUrl: "",
    fetcher: function fetcher() {
      return Promise.resolve({
        ok: false,
        status: 401,
        headers: new Map([["Content-Type", "application/json"]]),
        json: async () => ({ code: "unauthorized", message: "authentication is required" }),
      });
    },
  });
  let notified = 0;
  const stop = onAuthRequired(() => {
    notified += 1;
  });
  await assert.rejects(() => client.request("/api/v1/jobs"), (error) => error.status === 401);
  assert.equal(notified, 1);
  stop();
});
