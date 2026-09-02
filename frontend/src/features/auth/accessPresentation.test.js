import assert from "node:assert/strict";
import test from "node:test";

import {
  accessSummary,
  canInviteRole,
  companyLoginCopy,
  companyLoginPath,
  companySlugFromPath,
  loginSlugIssue,
  inviteRoleOptions,
  needsUsersCompanyPicker,
  pickDefaultCompanyId,
  resolveUsersCompanyId,
  roleLabel,
  usersCompanyPrompt,
} from "./accessPresentation.js";

test("roleLabel maps known roles to Russian labels", () => {
  assert.equal(roleLabel("purchaser"), "Закупщик");
  assert.equal(roleLabel("company_admin"), "Администратор компании");
  assert.equal(roleLabel("platform_admin"), "Администратор сервиса");
});

test("roleLabel falls back to a plain user label", () => {
  assert.equal(roleLabel("unknown"), "Пользователь");
});

test("inviteRoleOptions lists roles the actor may invite", () => {
  assert.deepEqual(inviteRoleOptions("platform_admin"), ["company_admin"]);
  assert.deepEqual(inviteRoleOptions("company_admin"), ["purchaser", "company_admin"]);
  assert.deepEqual(inviteRoleOptions("purchaser"), []);
});

test("canInviteRole follows inviteRoleOptions", () => {
  assert.equal(canInviteRole("company_admin", "purchaser"), true);
  assert.equal(canInviteRole("company_admin", "company_admin"), true);
  assert.equal(canInviteRole("platform_admin", "company_admin"), true);
  assert.equal(canInviteRole("platform_admin", "purchaser"), false);
  assert.equal(canInviteRole("platform_admin", "platform_admin"), false);
  assert.equal(canInviteRole("purchaser", "purchaser"), false);
});

test("accessSummary explains what each role can do", () => {
  assert.match(accessSummary("platform_admin"), /компани/i);
  assert.match(accessSummary("company_owner"), /сотрудник/i);
  assert.match(accessSummary("company_admin"), /приглаш/i);
  assert.match(accessSummary("purchaser"), /выгрузк/i);
});

test("platform admin must pick a company to manage users", () => {
  assert.equal(needsUsersCompanyPicker("platform_admin"), true);
  assert.equal(needsUsersCompanyPicker("company_admin"), false);
  assert.equal(needsUsersCompanyPicker("purchaser"), false);
});

test("resolveUsersCompanyId uses the selected company only for platform admin", () => {
  assert.equal(resolveUsersCompanyId("platform_admin", "c-1", ""), "c-1");
  assert.equal(resolveUsersCompanyId("platform_admin", "", ""), "");
  assert.equal(resolveUsersCompanyId("company_admin", "other", "own"), "own");
});

test("pickDefaultCompanyId keeps the current selection or takes the first active company", () => {
  const companies = [
    { id: "off", disabled_at: "2026-01-01" },
    { id: "a", disabled_at: "" },
    { id: "b", disabled_at: "" },
  ];
  assert.equal(pickDefaultCompanyId("b", companies), "b");
  assert.equal(pickDefaultCompanyId("", companies), "a");
  assert.equal(pickDefaultCompanyId("", [{ id: "off", disabled_at: "2026-01-01" }]), "");
});

test("usersCompanyPrompt tells platform admin why the list is empty", () => {
  assert.equal(usersCompanyPrompt("c-1", [{ id: "c-1" }]), "");
  assert.equal(usersCompanyPrompt("", []), "Сначала создайте компанию.");
  assert.equal(
    usersCompanyPrompt("", [{ id: "off", disabled_at: "2026-01-01" }]),
    "Сначала создайте компанию.",
  );
  assert.equal(
    usersCompanyPrompt("", [{ id: "a", disabled_at: "" }]),
    "Выберите компанию, чтобы увидеть сотрудников.",
  );
});

test("companySlugFromPath reads /c/:slug and leaves invite routes alone", () => {
  assert.equal(companySlugFromPath("/c/acme"), "acme");
  assert.equal(companySlugFromPath("/c/acme%20co"), "acme co");
  assert.equal(companySlugFromPath("/invite/token-1"), "");
  assert.equal(companySlugFromPath("/"), "");
});

test("companyLoginCopy personalizes the login screen without leaking ids", () => {
  assert.deepEqual(companyLoginCopy(null), { title: "Вход", lead: "" });
  assert.deepEqual(companyLoginCopy({ name: "Acme" }), {
    title: "Вход для сотрудников «Acme»",
    lead: "Работайте только с файлами своей компании.",
  });
});

test("loginSlugIssue requires latin letters digits and hyphen", () => {
  assert.equal(loginSlugIssue(""), "Укажите адрес входа латиницей.");
  assert.match(loginSlugIssue("Кристайл"), /латиниц/i);
  assert.match(loginSlugIssue("admin"), /зарезерв/i);
  assert.equal(loginSlugIssue("Kristail"), "");
});

test("companyLoginPath is a local /c/:slug link", () => {
  assert.equal(companyLoginPath("kristail"), "/c/kristail");
});
