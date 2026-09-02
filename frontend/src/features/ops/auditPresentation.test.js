import assert from "node:assert/strict";
import test from "node:test";

import { accessAuditEvents, auditLine } from "./auditPresentation.js";

test("auditLine turns access actions into Russian sentences", () => {
  assert.equal(
    auditLine({ action: "invite_created", actor_login: "анна", company_name: "Сияние" }),
    "анна пригласил человека в «Сияние»",
  );
  assert.equal(auditLine({ action: "password_changed", actor_login: "анна" }), "анна сменил пароль");
  assert.equal(auditLine({ action: "login_success", actor_login: "анна" }), "");
});

test("accessAuditEvents drops logins and keeps a readable line", () => {
  const items = accessAuditEvents([
    { id: "1", action: "login_success", actor_login: "анна" },
    { id: "2", action: "user_disabled", actor_login: "анна", company_name: "Сияние" },
  ]);
  assert.equal(items.length, 1);
  assert.equal(items[0].line, "анна отключил сотрудника в «Сияние»");
});
