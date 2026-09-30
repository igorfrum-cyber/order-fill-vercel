import test from "node:test";
import assert from "node:assert/strict";

import { findPreviewArticle, getPreviewMeta, getPreviewWindow } from "./preview.js";
import { apiClient } from "./client.js";

function stubClient() {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = async (url, options) => {
    calls.push({ url, options });
    return {
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({}),
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

test("getPreviewMeta escapes job and file identifiers", async () => {
  const stub = stubClient();
  try {
    await getPreviewMeta("job/1", "file 2");
    assert.equal(stub.calls[0].url, "/api/v1/jobs/job%2F1/files/file%202/preview");
  } finally {
    stub.restore();
  }
});

test("getPreviewWindow serializes the default and requested sheet range", async () => {
  const stub = stubClient();
  try {
    await getPreviewWindow("job", "file", { fromRow: 4, toRow: 9 });
    const url = new URL(stub.calls[0].url, "http://localhost");
    assert.equal(url.pathname, "/api/v1/jobs/job/files/file/preview/window");
    assert.equal(url.searchParams.get("sheet"), "0");
    assert.equal(url.searchParams.get("from_row"), "4");
    assert.equal(url.searchParams.get("to_row"), "9");
  } finally {
    stub.restore();
  }
});

test("findPreviewArticle serializes the sheet and search text", async () => {
  const stub = stubClient();
  try {
    await findPreviewArticle("job", "file", { sheet: 2, query: "A&B" });
    const url = new URL(stub.calls[0].url, "http://localhost");
    assert.equal(url.pathname, "/api/v1/jobs/job/files/file/preview/find");
    assert.equal(url.searchParams.get("sheet"), "2");
    assert.equal(url.searchParams.get("q"), "A&B");
  } finally {
    stub.restore();
  }
});
