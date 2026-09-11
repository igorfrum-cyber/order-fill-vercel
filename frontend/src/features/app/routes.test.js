import assert from "node:assert/strict";
import test from "node:test";

import { companyIdFromSearch, parseAppPath, pathForScreen, resolveOrderNavJobId, screenAllowed, withCompanyQuery } from "./routes.js";

test("parseAppPath reads signed-in screens and job ids", () => {
  assert.deepEqual(parseAppPath("/"), { screen: "", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/overview"), { screen: "overview", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/queue"), { screen: "queue", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/jobs"), { screen: "history", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/jobs/new"), { screen: "order", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/jobs/abc"), { screen: "order", jobId: "abc", unknown: false });
  assert.deepEqual(parseAppPath("/company"), { screen: "company", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/users"), { screen: "users", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/account"), { screen: "account", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/companies"), { screen: "companies", jobId: "", unknown: false });
  assert.equal(parseAppPath("/nope").unknown, true);
});

test("parseAppPath ignores invite and company login paths", () => {
  assert.equal(parseAppPath("/invite/tok").screen, "");
  assert.equal(parseAppPath("/c/acme").screen, "");
});

test("pathForScreen writes the URL for a screen", () => {
  assert.equal(pathForScreen("overview"), "/overview");
  assert.equal(pathForScreen("queue"), "/queue");
  assert.equal(pathForScreen("history"), "/jobs");
  assert.equal(pathForScreen("order"), "/jobs/new");
  assert.equal(pathForScreen("order", "abc"), "/jobs/abc");
  assert.equal(pathForScreen("company"), "/company");
});

test("resolveOrderNavJobId keeps the open job when Work is clicked again", () => {
  assert.equal(resolveOrderNavJobId("order", "", { screen: "order", openJobId: "abc" }), "abc");
  assert.equal(resolveOrderNavJobId("order", "", { screen: "history", openJobId: "abc" }), "");
  assert.equal(resolveOrderNavJobId("order", "xyz", { screen: "order", openJobId: "abc" }), "xyz");
});

test("screenAllowed rejects foreign screens", () => {
  assert.equal(screenAllowed("purchaser", "users"), false);
  assert.equal(screenAllowed("purchaser", "order"), true);
  assert.equal(screenAllowed("platform_admin", "order"), false);
  assert.equal(screenAllowed("platform_admin", "overview"), true);
  assert.equal(screenAllowed("company_admin", "company"), true);
  assert.equal(screenAllowed("company_admin", "queue"), true);
  assert.equal(screenAllowed("purchaser", "queue"), false);
  assert.equal(screenAllowed("platform_admin", "order", { jobId: "abc" }), true);
});

test("companyQuery reads and writes platform company id", () => {
  assert.equal(companyIdFromSearch("?company=c-1"), "c-1");
  assert.equal(withCompanyQuery("/jobs", "c-1"), "/jobs?company=c-1");
  assert.equal(withCompanyQuery("/jobs", ""), "/jobs");
});
