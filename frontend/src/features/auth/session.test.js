import assert from "node:assert/strict";
import test from "node:test";

import { formatSessionWhen, passkeyWhen, sessionIsPhone } from "./session.js";

test("sessionIsPhone detects phones and tablets", () => {
  assert.equal(sessionIsPhone("Chrome на iPhone"), true);
  assert.equal(sessionIsPhone("Safari на iPad"), true);
  assert.equal(sessionIsPhone("Chrome на Android"), true);
  assert.equal(sessionIsPhone("Safari на Mac"), false);
  assert.equal(sessionIsPhone("Chrome на Windows"), false);
});

test("formatSessionWhen uses a short Russian timestamp", () => {
  assert.equal(formatSessionWhen(""), "");
  assert.match(formatSessionWhen("2026-09-02T15:31:00Z"), /сент/i);
});

test("passkeyWhen prefers last used over created", () => {
  assert.match(passkeyWhen({ last_used_at: "2026-09-02T15:31:00Z" }), /вход/i);
  assert.match(passkeyWhen({ created_at: "2026-09-01T10:00:00Z" }), /добавлен/i);
  assert.equal(passkeyWhen({}), "");
});
