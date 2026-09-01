import test from "node:test";
import assert from "node:assert/strict";

import { changePassword, login } from "./auth.js";
import { ApiClient, apiClient } from "./client.js";

test("login posts credentials with CSRF header and cookies", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options, receiver: this });
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ id: "u1", login: "buyer", role: "purchaser" }),
    });
  };
  try {
    const user = await login("buyer", "correct-horse");
    assert.equal(user.login, "buyer");
    assert.equal(calls[0].receiver, globalThis);
    assert.equal(calls[0].options.credentials, "include");
    assert.equal(calls[0].options.headers["X-Requested-With"], "fetch");
    assert.match(calls[0].options.body, /correct-horse/);
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("changePassword posts current and next password", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({ ok: true, status: 204, headers: new Map() });
  };
  try {
    await changePassword("old-password-1", "new-password-1");
    assert.equal(calls[0].url, "/api/v1/auth/password");
    assert.match(calls[0].options.body, /current_password/);
    assert.match(calls[0].options.body, /new-password-1/);
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("ApiClient.requestDownload sends credentials", async () => {
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
  const client = new ApiClient({
    baseUrl: "http://api.example",
    fetcher: function fetcher(url, options) {
      assert.equal(options.credentials, "include");
      assert.equal(url, "http://api.example/api/v1/jobs/job-1/archive");
      return Promise.resolve({ ok: true, status: 200, headers, blob: async () => blob });
    },
  });
  const download = await client.requestDownload("/api/v1/jobs/job-1/archive");
  assert.equal(download.fileName, "angiopharm_2026-09.zip");
});
