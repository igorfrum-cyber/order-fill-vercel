import test from "node:test";
import assert from "node:assert/strict";

import { ApiClient, onAuthRequired } from "./client.js";

function headers(values = {}) {
  return { get: (name) => values[name] ?? null };
}

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

test("ApiClient.absoluteUrl handles empty, absolute and relative paths", () => {
  const client = new ApiClient({ baseUrl: "http://api.example///", fetcher: async () => {} });
  assert.equal(client.absoluteUrl(""), "");
  assert.equal(client.absoluteUrl("http://cdn.example/file"), "http://cdn.example/file");
  assert.equal(client.absoluteUrl("files/1"), "http://api.example/files/1");
  assert.equal(client.absoluteUrl("/files/1"), "http://api.example/files/1");
});

test("ApiClient.request handles no-content and non-JSON responses", async () => {
  const blob = new Blob(["binary"]);
  const responses = [
    { ok: true, status: 204, headers: headers() },
    { ok: true, status: 200, headers: headers({ "Content-Type": "application/octet-stream" }), blob: async () => blob },
  ];
  const client = new ApiClient({ baseUrl: "", fetcher: async () => responses.shift() });

  assert.equal(await client.request("/empty"), null);
  assert.equal(await client.request("/binary"), blob);
});

test("ApiClient.request sets JSON headers but leaves multipart content type to fetch", async () => {
  const calls = [];
  const client = new ApiClient({
    baseUrl: "",
    fetcher: async (url, options) => {
      calls.push({ url, options });
      return { ok: true, status: 200, headers: headers(), json: async () => null, blob: async () => new Blob() };
    },
  });

  await client.request("/json", { headers: { "X-Test": "yes" } });
  const body = new FormData();
  body.append("file", "a.xlsx");
  await client.request("/upload", { method: "POST", body });

  assert.equal(calls[0].options.headers["Content-Type"], "application/json");
  assert.equal(calls[0].options.headers["X-Test"], "yes");
  assert.equal(calls[1].options.headers["Content-Type"], undefined);
  assert.equal(calls[1].options.method, "POST");
});

test("ApiClient reports non-JSON request errors and does not notify auth routes", async () => {
  const client = new ApiClient({
    baseUrl: "",
    fetcher: async (url) => ({
      ok: false,
      status: 401,
      headers: headers({ "Content-Type": "text/plain" }),
      text: async () => `expired at ${url}`,
    }),
  });
  let notified = 0;
  const stop = onAuthRequired(() => { notified += 1; });
  try {
    for (const path of ["/api/v1/auth/login", "/api/v1/auth/invite/accept", "/api/v1/auth/passkeys/login/begin"]) {
      await assert.rejects(client.request(path), (error) => {
        assert.equal(error.status, 401);
        assert.equal(error.payload.message, `expired at ${path}`);
        return true;
      });
    }
    assert.equal(notified, 0);
  } finally {
    stop();
  }
});

test("ApiClient.requestDownload reports missing content headers and HTTP errors", async () => {
  const responses = [
    { ok: true, status: 200, headers: headers(), blob: async () => new Blob(["raw"]) },
    { ok: false, status: 500, headers: headers({ "Content-Type": "text/plain" }), text: async () => "storage unavailable" },
  ];
  const client = new ApiClient({ baseUrl: "", fetcher: async () => responses.shift() });
  const download = await client.requestDownload("/file");
  assert.equal(download.contentType, "application/octet-stream");
  assert.equal(download.fileName, "");
  assert.equal(download.blob.size, 3);
  await assert.rejects(client.requestDownload("/file"), (error) => {
    assert.equal(error.status, 500);
    assert.equal(error.payload.message, "storage unavailable");
    return true;
  });
});
