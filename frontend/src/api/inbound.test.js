import test from "node:test";
import assert from "node:assert/strict";

import { downloadInboundFile, getInboundCompany, getInboundDeliveries, getInboundMessages, getInboundSettings, updateInboundCompany, updateInboundSettings } from "./inbound.js";
import { apiClient } from "./client.js";

function stubClient(jsonFor) {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = async (url, options) => {
    calls.push({ url, options });
    return {
      ok: true,
      status: options?.method === "POST" ? 200 : 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => (typeof jsonFor === "function" ? jsonFor(url, options) : jsonFor),
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

test("inbound API fetches settings and posts the enabled flag", async () => {
  const stub = stubClient(() => ({ enabled: false, webhook_count: 3, error_count: 0 }));
  try {
    await getInboundSettings();
    await updateInboundSettings({ enabled: true });
    assert.equal(stub.calls[0].url, "/api/v1/inbound/settings");
    assert.equal(stub.calls[1].url, "/api/v1/inbound/settings");
    assert.equal(stub.calls[1].options.method, "POST");
    assert.equal(JSON.parse(stub.calls[1].options.body).enabled, true);
  } finally {
    stub.restore();
  }
});

test("inbound API reads and updates a company address", async () => {
  const stub = stubClient(() => ({ company_id: "c1", receive_address: "zakaz@example.com" }));
  try {
    await getInboundCompany("c/1");
    await updateInboundCompany("c/1", { receive_address: "new@example.com", allowed_from: ["1c@example.com"], enabled: true });
    assert.equal(stub.calls[0].url, "/api/v1/inbound/companies/c%2F1");
    assert.equal(stub.calls[1].url, "/api/v1/inbound/companies/c%2F1");
    assert.equal(stub.calls[1].options.method, "POST");
    assert.deepEqual(JSON.parse(stub.calls[1].options.body), { receive_address: "new@example.com", allowed_from: ["1c@example.com"], enabled: true });
  } finally {
    stub.restore();
  }
});

test("inbound API lists messages and deliveries", async () => {
  const stub = stubClient(() => ({ messages: [] }));
  try {
    await getInboundCompany("c1");
    await getInboundMessages("c1");
    await getInboundDeliveries();
    assert.equal(stub.calls[1].url, "/api/v1/inbound/companies/c1/messages");
    assert.equal(stub.calls[2].url, "/api/v1/inbound/deliveries");
  } finally {
    stub.restore();
  }
});

test("inbound API downloads an attachment with credentials", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = async (url, options) => {
    calls.push({ url, options });
    return {
      ok: true,
      status: 200,
      headers: new Map([
        ["Content-Type", "application/vnd.ms-excel"],
        ["Content-Disposition", "attachment; filename*=UTF-8''%D0%B7%D0%B0%D1%8F%D0%B2%D0%BA%D0%B0.xlsx"],
      ]),
      blob: async () => new Blob(["data"]),
    };
  };
  try {
    const download = await downloadInboundFile("c1", "msg-1", "att-2");
    assert.equal(calls[0].url, "/api/v1/inbound/companies/c1/messages/msg-1/files/att-2");
    assert.equal(calls[0].options.credentials, "include");
    assert.equal(download.fileName, "заявка.xlsx");
    assert.equal(download.contentType, "application/vnd.ms-excel");
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});