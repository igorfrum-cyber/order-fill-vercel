import test from "node:test";
import assert from "node:assert/strict";

import { isPollDone } from "./jobs.js";

test("isPollDone treats needs_review as finished during the first fill", () => {
  assert.equal(isPollDone("needs_review"), true);
  assert.equal(isPollDone("completed"), true);
  assert.equal(isPollDone("failed"), true);
  assert.equal(isPollDone("processing"), false);
  assert.equal(isPollDone("finalizing"), false);
});

test("isPollDone waits past needs_review when finalizing reviewer edits", () => {
  const until = ["completed", "failed"];
  assert.equal(isPollDone("needs_review", until), false);
  assert.equal(isPollDone("finalizing", until), false);
  assert.equal(isPollDone("processing", until), false);
  assert.equal(isPollDone("completed", until), true);
  assert.equal(isPollDone("failed", until), true);
});
