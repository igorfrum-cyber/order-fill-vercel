import assert from "node:assert/strict";
import test from "node:test";

import {
  accessSummary,
  canEditCompanyProfile,
  canInviteRole,
  canManageListedUser,
  companyLoginLogoURL,
  companyLoginCopy,
  companyLoginPath,
  companyLoginURL,
  loginParentHost,
  companySlugFromHost,
  companySlugFromPath,
  loginSlugIssue,
  inviteRoleHint,
  inviteRoleOptions,
  needsSecurityNudge,
  homeScreen,
  needsUsersCompanyPicker,
  pickDefaultCompanyId,
  resolveUsersCompanyId,
  roleLabel,
  usersCompanyPrompt,
} from "./accessPresentation.js";

test("roleLabel maps known roles to Russian labels", () => {
  assert.equal(roleLabel("purchaser"), "Закупщик");
  assert.equal(roleLabel("company_admin"), "Администратор компании");
  assert.equal(roleLabel("company_owner"), "Владелец компании");
  assert.equal(roleLabel("platform_admin"), "Администратор сервиса");
});

test("roleLabel falls back to a plain user label", () => {
  assert.equal(roleLabel("unknown"), "Пользователь");
});

test("inviteRoleOptions lists roles the actor may invite", () => {
  assert.deepEqual(inviteRoleOptions("platform_admin"), ["company_owner", "company_admin", "purchaser"]);
  assert.deepEqual(inviteRoleOptions("company_owner"), ["company_admin", "purchaser"]);
  assert.deepEqual(inviteRoleOptions("company_admin"), ["purchaser"]);
  assert.deepEqual(inviteRoleOptions("purchaser"), []);
});

test("canInviteRole follows inviteRoleOptions", () => {
  assert.equal(canInviteRole("platform_admin", "company_owner"), true);
  assert.equal(canInviteRole("platform_admin", "company_admin"), true);
  assert.equal(canInviteRole("platform_admin", "purchaser"), true);
  assert.equal(canInviteRole("platform_admin", "platform_admin"), false);
  assert.equal(canInviteRole("company_owner", "company_admin"), true);
  assert.equal(canInviteRole("company_owner", "purchaser"), true);
  assert.equal(canInviteRole("company_owner", "company_owner"), false);
  assert.equal(canInviteRole("company_admin", "purchaser"), true);
  assert.equal(canInviteRole("company_admin", "company_admin"), false);
  assert.equal(canInviteRole("company_admin", "company_owner"), false);
  assert.equal(canInviteRole("purchaser", "purchaser"), false);
});

test("company admin cannot disable or reset the company owner", () => {
  assert.equal(canManageListedUser("platform_admin", "company_owner"), true);
  assert.equal(canManageListedUser("company_owner", "company_admin"), true);
  assert.equal(canManageListedUser("company_owner", "purchaser"), true);
  assert.equal(canManageListedUser("company_admin", "purchaser"), true);
  assert.equal(canManageListedUser("company_admin", "company_owner"), false);
  assert.equal(canManageListedUser("purchaser", "purchaser"), false);
});

test("accessSummary explains what each role can do", () => {
  assert.match(accessSummary("platform_admin"), /компани/i);
  assert.match(accessSummary("company_owner"), /сотрудник/i);
  assert.match(accessSummary("company_admin"), /приглаш/i);
  assert.match(accessSummary("purchaser"), /выгрузк/i);
});

test("inviteRoleHint explains the dropdown without jargon", () => {
  assert.equal(
    inviteRoleHint,
    "Выберите, что человек сможет делать в компании. Доступ можно отключить позже.",
  );
});

test("company owner and company admin edit the company profile", () => {
  assert.equal(canEditCompanyProfile("company_owner"), true);
  assert.equal(canEditCompanyProfile("company_admin"), true);
  assert.equal(canEditCompanyProfile("platform_admin"), false);
  assert.equal(canEditCompanyProfile("purchaser"), false);
});

test("needsSecurityNudge prompts every signed-in user until a passkey or code is on", () => {
  assert.equal(needsSecurityNudge({ role: "purchaser" }), true);
  assert.equal(needsSecurityNudge({ role: "company_owner" }), true);
  assert.equal(needsSecurityNudge({ role: "company_admin" }), true);
  assert.equal(needsSecurityNudge({ role: "platform_admin" }), true);
  assert.equal(needsSecurityNudge({ role: "purchaser", two_factor_enabled: true }), false);
  assert.equal(needsSecurityNudge({ role: "purchaser", has_passkey: true }), false);
  assert.equal(needsSecurityNudge({ role: "company_owner", two_factor_enabled: true }), false);
});

test("homeScreen lands company users on history and platform admin on overview", () => {
  assert.equal(homeScreen("platform_admin"), "overview");
  assert.equal(homeScreen("company_owner"), "history");
  assert.equal(homeScreen("company_admin"), "history");
  assert.equal(homeScreen("purchaser"), "history");
});

test("platform admin must pick a company to manage users", () => {
  assert.equal(needsUsersCompanyPicker("platform_admin"), true);
  assert.equal(needsUsersCompanyPicker("company_owner"), false);
  assert.equal(needsUsersCompanyPicker("company_admin"), false);
  assert.equal(needsUsersCompanyPicker("purchaser"), false);
});

test("resolveUsersCompanyId uses the selected company only for platform admin", () => {
  assert.equal(resolveUsersCompanyId("platform_admin", "c-1", ""), "c-1");
  assert.equal(resolveUsersCompanyId("platform_admin", "", ""), "");
  assert.equal(resolveUsersCompanyId("company_owner", "other", "own"), "own");
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

test("companyLoginLogoURL is a public slug path", () => {
  assert.equal(companyLoginLogoURL("kristail"), "/api/v1/public/companies/kristail/logo");
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

test("companySlugFromHost reads the first label of a company subdomain", () => {
  assert.equal(companySlugFromHost("kristail.localhost"), "kristail");
  assert.equal(companySlugFromHost("kristail.localhost:3200"), "kristail");
  assert.equal(companySlugFromHost("localhost"), "");
  assert.equal(companySlugFromHost("127.0.0.1"), "");
  assert.equal(companySlugFromHost("www.example.com"), "");
});

test("companyLoginURL builds a localhost subdomain for local hosts", () => {
  assert.equal(
    companyLoginURL("kristail", { protocol: "http:", hostname: "127.0.0.1", port: "3200" }),
    "http://kristail.localhost:3200/",
  );
  assert.equal(
    companyLoginURL("kristail", { protocol: "http:", hostname: "localhost", port: "3200" }),
    "http://kristail.localhost:3200/",
  );
  assert.equal(
    companyLoginURL("chernovaa", { protocol: "https:", hostname: "example.com", port: "" }),
    "https://chernovaa.example.com/",
  );
  assert.equal(
    companyLoginURL("art72", { protocol: "https:", hostname: "art72.example.com", port: "" }),
    "https://art72.example.com/",
  );
});

test("companyLoginURL uses a path on a public IP because subdomains cannot attach to an address", () => {
  assert.equal(
    companyLoginURL("art72", { protocol: "http:", hostname: "203.0.113.10", port: "3200" }),
    "http://203.0.113.10:3200/c/art72",
  );
  assert.equal(
    companyLoginURL("art72", { protocol: "http:", hostname: "192.168.31.108", port: "" }),
    "http://192.168.31.108/c/art72",
  );
});

test("companyLoginURL uses a path on DuckDNS because a wildcard certificate is not assumed", () => {
  assert.equal(
    companyLoginURL("art72", { protocol: "https:", hostname: "orderfill.duckdns.org", port: "" }),
    "https://orderfill.duckdns.org/c/art72",
  );
  assert.equal(
    companyLoginURL("art72", { protocol: "https:", hostname: "203-0-113-10.sslip.io", port: "" }),
    "https://203-0-113-10.sslip.io/c/art72",
  );
});

test("loginParentHost does not collapse a DuckDNS name onto duckdns.org", () => {
  assert.equal(loginParentHost("orderfill.duckdns.org"), "orderfill.duckdns.org");
  assert.equal(loginParentHost("art72.orderfill.duckdns.org"), "orderfill.duckdns.org");
  assert.equal(loginParentHost("art72.example.com"), "example.com");
});
