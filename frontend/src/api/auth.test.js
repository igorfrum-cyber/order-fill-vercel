import test from "node:test";
import assert from "node:assert/strict";

import { changePassword, completeTwoFactorLogin, createCompany, disableTwoFactor, enableTwoFactor, getCompanyLogin, listAudit, listSessions, listStatus, login, logoutEverywhere, setCompanyLogo, setCompanyLoginSlug, startTwoFactorSetup, updateCompany } from "./auth.js";
import { ApiClient, apiClient } from "./client.js";

test("getCompanyLogin requests public company metadata and encodes the slug", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ name: "Acme", login_slug: "acme" }),
    });
  };
  try {
    const company = await getCompanyLogin("acme/x");
    assert.equal(company.name, "Acme");
    assert.equal(calls[0].url, "/api/v1/public/companies/acme%2Fx/login");
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("createCompany posts name and latin login slug", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({
      ok: true,
      status: 201,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ id: "c1", name: "Кристайл", login_slug: "kristail" }),
    });
  };
  try {
    const company = await createCompany("Кристайл", "kristail");
    assert.equal(company.login_slug, "kristail");
    assert.equal(calls[0].url, "/api/v1/companies");
    assert.match(calls[0].options.body, /login_slug/);
    assert.match(calls[0].options.body, /kristail/);
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("setCompanyLoginSlug posts the new latin address", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ id: "c1", name: "Кристайл", login_slug: "kristail" }),
    });
  };
  try {
    await setCompanyLoginSlug("c1", "kristail");
    assert.equal(calls[0].url, "/api/v1/companies/c1/login-slug");
    assert.equal(calls[0].options.method, "POST");
    assert.match(calls[0].options.body, /kristail/);
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("updateCompany posts name and latin login slug", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ id: "c1", name: "Кристайл", login_slug: "kristail" }),
    });
  };
  try {
    const company = await updateCompany("c1", "Кристайл", "kristail");
    assert.equal(company.name, "Кристайл");
    assert.equal(calls[0].url, "/api/v1/companies/c1/profile");
    assert.equal(calls[0].options.method, "POST");
    assert.match(calls[0].options.body, /Кристайл/);
    assert.match(calls[0].options.body, /kristail/);
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("setCompanyLogo posts the image as multipart", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ id: "c1", has_logo: true }),
    });
  };
  try {
    const file = new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], "logo.png", { type: "image/png" });
    const company = await setCompanyLogo("c1", file);
    assert.equal(company.has_logo, true);
    assert.equal(calls[0].url, "/api/v1/companies/c1/logo");
    assert.equal(calls[0].options.method, "POST");
    assert.equal(calls[0].options.body instanceof FormData, true);
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

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

test("completeTwoFactorLogin posts challenge id and code", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ id: "u1", login: "buyer", role: "purchaser" }),
    });
  };
  try {
    const user = await completeTwoFactorLogin("challenge-1", "123456");
    assert.equal(user.login, "buyer");
    assert.equal(calls[0].url, "/api/v1/auth/login/2fa");
    assert.match(calls[0].options.body, /challenge_id/);
    assert.match(calls[0].options.body, /123456/);
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("two-factor setup enable and disable post the expected bodies", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    if (url.endsWith("/disable")) {
      return Promise.resolve({ ok: true, status: 204, headers: new Map() });
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => (url.endsWith("/setup") ? { secret: "ABC", otpauth_url: "otpauth://", qr_png_base64: "png" } : { recovery_codes: ["AAAA-BBBB"] }),
    });
  };
  try {
    const setup = await startTwoFactorSetup();
    assert.equal(setup.secret, "ABC");
    assert.equal(calls[0].url, "/api/v1/auth/2fa/setup");
    const enabled = await enableTwoFactor("123456");
    assert.deepEqual(enabled.recovery_codes, ["AAAA-BBBB"]);
    assert.equal(calls[1].url, "/api/v1/auth/2fa/enable");
    assert.match(calls[1].options.body, /123456/);
    await disableTwoFactor("correct-horse");
    assert.equal(calls[2].url, "/api/v1/auth/2fa/disable");
    assert.match(calls[2].options.body, /correct-horse/);
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

test("logoutEverywhere posts to the all-sessions endpoint", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({ ok: true, status: 204, headers: new Map() });
  };
  try {
    await logoutEverywhere();
    assert.equal(calls[0].url, "/api/v1/auth/logout-everywhere");
    assert.equal(calls[0].options.method, "POST");
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("listSessions reads active login devices", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url, options) {
    calls.push({ url, options });
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => ({ sessions: [{ id: "here", device: "Safari на Mac", current: true }] }),
    });
  };
  try {
    const result = await listSessions();
    assert.equal(calls[0].url, "/api/v1/auth/sessions");
    assert.equal(result.sessions[0].device, "Safari на Mac");
  } finally {
    apiClient.fetcher = originalFetcher;
    apiClient.baseUrl = originalBase;
  }
});

test("listStatus and listAudit hit the platform ops endpoints", async () => {
  const calls = [];
  const originalFetcher = apiClient.fetcher;
  const originalBase = apiClient.baseUrl;
  apiClient.baseUrl = "";
  apiClient.fetcher = function fetcher(url) {
    calls.push(url);
    return Promise.resolve({
      ok: true,
      status: 200,
      headers: new Map([["Content-Type", "application/json"]]),
      json: async () => (url.includes("status") ? { components: [] } : { events: [] }),
    });
  };
  try {
    await listStatus();
    await listAudit();
    assert.deepEqual(calls, ["/api/v1/status", "/api/v1/audit"]);
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
