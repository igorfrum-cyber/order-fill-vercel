import assert from "node:assert/strict";
import test from "node:test";

import { accessSummary, canInviteRole, inviteRoleOptions, roleLabel } from "./accessPresentation.js";

test("roleLabel maps known roles to Russian labels", () => {
  assert.equal(roleLabel("purchaser"), "Закупщик");
  assert.equal(roleLabel("company_admin"), "Администратор компании");
  assert.equal(roleLabel("platform_admin"), "Администратор сервиса");
});

test("roleLabel falls back to a plain user label", () => {
  assert.equal(roleLabel("unknown"), "Пользователь");
});

test("inviteRoleOptions lists roles the actor may invite", () => {
  assert.deepEqual(inviteRoleOptions("platform_admin"), ["purchaser", "company_admin"]);
  assert.deepEqual(inviteRoleOptions("company_admin"), ["purchaser", "company_admin"]);
  assert.deepEqual(inviteRoleOptions("purchaser"), []);
});

test("canInviteRole follows inviteRoleOptions", () => {
  assert.equal(canInviteRole("company_admin", "purchaser"), true);
  assert.equal(canInviteRole("company_admin", "company_admin"), true);
  assert.equal(canInviteRole("platform_admin", "platform_admin"), false);
  assert.equal(canInviteRole("purchaser", "purchaser"), false);
});

test("accessSummary explains what each role can do", () => {
  assert.match(accessSummary("platform_admin"), /компани/i);
  assert.match(accessSummary("company_admin"), /приглаш/i);
  assert.match(accessSummary("purchaser"), /выгрузк/i);
});
