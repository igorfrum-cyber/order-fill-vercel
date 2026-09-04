import assert from "node:assert/strict";
import test from "node:test";

import {
  consumeQuickStart,
  dismissQuickStart,
  quickStartKey,
  shouldAutoStartTour,
  shouldShowQuickStart,
  tourSceneForView,
} from "./firstRun.js";

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

test("home tour auto-starts only after invite, not on later logins", () => {
  assert.equal(shouldAutoStartTour("invite"), true);
  assert.equal(shouldAutoStartTour("login"), false);
  assert.equal(shouldAutoStartTour("session"), false);
});

test("tourSceneForView maps each working screen to its own hints", () => {
  assert.equal(tourSceneForView({ screen: "history" }), "home");
  assert.equal(tourSceneForView({ screen: "overview" }), "home");
  assert.equal(tourSceneForView({ screen: "overview", seenHome: true }), "overview");
  assert.equal(tourSceneForView({ screen: "users" }), "users");
  assert.equal(tourSceneForView({ screen: "company" }), "company");
  assert.equal(tourSceneForView({ screen: "companies" }), "companies");
  assert.equal(tourSceneForView({ screen: "account" }), "account");
  assert.equal(tourSceneForView({ screen: "order", stage: "upload" }), "order-upload");
  assert.equal(tourSceneForView({ screen: "order", stage: "fill" }), "order-fill");
  assert.equal(tourSceneForView({ screen: "order", stage: "preview" }), "order-preview");
  assert.equal(tourSceneForView({ screen: "north" }), "north");
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
