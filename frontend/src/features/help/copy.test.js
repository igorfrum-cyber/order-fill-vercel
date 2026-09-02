import assert from "node:assert/strict";
import test from "node:test";

import { helpSections, loginAccessHint, loginFailedMessage, quickStartForRole, roleLabel } from "./copy.js";

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
