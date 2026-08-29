import test from "node:test";
import assert from "node:assert/strict";

import { ApiClient } from "./client.js";

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
