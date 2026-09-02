import assert from "node:assert/strict";
import test from "node:test";

import { consumeQuickStart, dismissQuickStart, quickStartKey, shouldShowQuickStart } from "./firstRun.js";

test("quick start key is scoped by user id", () => {
  assert.equal(quickStartKey({ id: "u1" }), "order-fill:quick-start:u1");
});

test("quick start shows until it has been seen once", () => {
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

test("consumeQuickStart opens once then stays closed", () => {
  const storage = new Map();
  const adapter = {
    getItem: (key) => storage.get(key) || null,
    setItem: (key, value) => {
      storage.set(key, value);
    },
  };
  assert.equal(consumeQuickStart({ id: "u1" }, adapter), true);
  assert.equal(consumeQuickStart({ id: "u1" }, adapter), false);
  assert.equal(shouldShowQuickStart({ id: "u1" }, adapter), false);
});
