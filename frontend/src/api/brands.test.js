import test from "node:test";
import assert from "node:assert/strict";

import { listBrandRules, updateBrandRule } from "./brands.js";
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
      json: async () => ({ rules: [] }),
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

test("listBrandRules requests the brand rules collection", async () => {
  const stub = stubClient();
  try {
    await listBrandRules();
    assert.equal(stub.calls[0].url, "/api/v1/brand-rules");
    assert.equal(stub.calls[0].options.method, undefined);
  } finally {
    stub.restore();
  }
});

test("updateBrandRule encodes the brand path and posts the rule", async () => {
  const stub = stubClient();
  try {
    const rule = { active: true, aliases: ["Новый бренд"] };
    await updateBrandRule("brand/name", rule);
    assert.equal(stub.calls[0].url, "/api/v1/brand-rules/brand%2Fname");
    assert.equal(stub.calls[0].options.method, "POST");
    assert.deepEqual(JSON.parse(stub.calls[0].options.body), rule);
  } finally {
    stub.restore();
  }
});
