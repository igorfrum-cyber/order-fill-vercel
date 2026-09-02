import assert from "node:assert/strict";
import test from "node:test";

import { lastSeenLabel, userInitial, usersByHierarchy } from "./userHierarchy.js";

test("usersByHierarchy splits a company into three bands", () => {
  const bands = usersByHierarchy([
    { id: "1", login: "owner", role: "company_owner" },
    { id: "2", login: "admin", role: "company_admin" },
    { id: "3", login: "buyer", role: "purchaser" },
    { id: "4", login: "other", role: "purchaser" },
  ]);
  assert.deepEqual(
    bands.map((band) => [band.key, band.users.map((user) => user.login)]),
    [
      ["owners", ["owner"]],
      ["admins", ["admin"]],
      ["purchasers", ["buyer", "other"]],
    ],
  );
});

test("lastSeenLabel is plain Russian", () => {
  assert.equal(lastSeenLabel(""), "Ещё не входил");
  assert.equal(lastSeenLabel(null), "Ещё не входил");
  const today = new Date();
  today.setHours(14, 32, 0, 0);
  assert.match(lastSeenLabel(today.toISOString()), /Вход сегодня,/);
  assert.match(lastSeenLabel("2020-01-15T08:15:00Z"), /Вход /);
});

test("userInitial uses the first letter of the login", () => {
  assert.equal(userInitial("анна"), "А");
  assert.equal(userInitial(""), "?");
});
