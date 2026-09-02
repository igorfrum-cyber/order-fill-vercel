import assert from "node:assert/strict";
import test from "node:test";

import { dismissQuickStart, quickStartKey, shouldShowQuickStart } from "./firstRun.js";

test("quick start key is scoped by user id", () => {
  assert.equal(quickStartKey({ id: "u1" }), "order-fill:quick-start:u1");
});

test("quick start shows until dismissed", () => {
  const storage = new Map();
  const adapter = { getItem: (key) => storage.get(key) || null };
  assert.equal(shouldShowQuickStart({ id: "u1" }, adapter), true);
});

test("quick start hides after dismiss", () => {
  const storage = new Map();
  const adapter = {
    getItem: (key) => storage.get(key) || null,
    setItem: (key, value) => {
      storage.set(key, value);
    },
  };
  dismissQuickStart({ id: "u1" }, adapter);
  assert.equal(shouldShowQuickStart({ id: "u1" }, adapter), false);
});
