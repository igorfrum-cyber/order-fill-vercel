import assert from "node:assert/strict";
import test from "node:test";

import { helpSections, loginAccessHint, loginFailedMessage, quickStartForRole, roleLabel, tourForRole } from "./copy.js";

test("loginFailedMessage does not distinguish why login failed", () => {
  assert.equal(
    loginFailedMessage,
    "Не получилось войти. Проверьте логин и пароль или запросите новую ссылку.",
  );
  assert.equal(/не найден|отключен|неверн|компани/i.test(loginFailedMessage), false);
});

test("loginAccessHint tells the user who can restore access", () => {
  assert.equal(
    loginAccessHint,
    "Нет доступа? Попросите владельца или администратора компании прислать приглашение.",
  );
});

test("roleLabel uses plain Russian labels", () => {
  assert.equal(roleLabel("company_admin"), "Администратор компании");
  assert.equal(roleLabel("platform_admin"), "Администратор сервиса");
  assert.equal(roleLabel("purchaser"), "Закупщик");
});

test("quickStartForRole returns non-technical steps", () => {
  const steps = quickStartForRole("purchaser");
  assert.ok(steps.length >= 3);
  assert.ok(steps.every((step) => !/api|token|cookie|backend|frontend|company_admin/i.test(step)));
});

test("tourForRole points at on-screen controls without jargon", () => {
  const known = new Set(["order", "north", "jobs", "help", "users", "companies", "company-select"]);
  const purchaser = tourForRole("purchaser");
  assert.deepEqual(
    purchaser.map((step) => step.target),
    ["order", "north", "jobs", "help"],
  );
  for (const role of ["purchaser", "company_admin", "company_owner", "platform_admin"]) {
    const steps = tourForRole(role);
    assert.ok(steps.length >= 3);
    for (const step of steps) {
      assert.ok(known.has(step.target), step.target);
      assert.ok(step.title);
      assert.ok(step.body);
      assert.equal(/api|token|cookie|backend|frontend|endpoint/i.test(`${step.title} ${step.body}`), false);
    }
  }
});

test("helpSections stay plain and cover the required topics", () => {
  const titles = helpSections.map((section) => section.title);
  assert.deepEqual(titles, [
    "Как сделать выгрузку",
    "Какие файлы нужны",
    'Что значит "Нужно проверить"',
    "Пользователи и доступ",
    "Если не получается войти",
  ]);
  const text = helpSections.map((section) => `${section.title} ${section.body}`).join("\n");
  assert.equal(/api|token|cookie|backend|frontend|endpoint/i.test(text), false);
  assert.ok(helpSections.every((section) => section.body.split(/(?<=[.!?])\s+/).length <= 2));
});
