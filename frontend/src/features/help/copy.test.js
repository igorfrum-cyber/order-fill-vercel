import assert from "node:assert/strict";
import test from "node:test";

import {
  accessSummaryForRole,
  accountPasswordHint,
  helpSections,
  loginAccessHint,
  loginFailedMessage,
  logoutEverywhereConfirm,
  logoutEverywhereLabel,
  headerContext,
  profileCompanyLabel,
  profileFields,
  quickStartForRole,
  roleLabel,
  tourForRole,
  inviteRoleHint,
  twoFactorCodeLabel,
  twoFactorOpenAppLabel,
  twoFactorRecoveryHint,
  twoFactorSetupSteps,
  securitySetupLabel,
  twoFactorEnableLabel,
  twoFactorLoginTitle,
  twoFactorRequiredHint,
  twoFactorSetupHint,
  passkeyInsecureOriginHint,
} from "./copy.js";

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
  assert.equal(roleLabel("company_owner"), "Владелец компании");
  assert.equal(roleLabel("company_admin"), "Администратор компании");
  assert.equal(roleLabel("platform_admin"), "Администратор сервиса");
  assert.equal(roleLabel("purchaser"), "Закупщик");
});

test("inviteRoleHint explains the company access choice", () => {
  assert.equal(
    inviteRoleHint,
    "Выберите, что человек сможет делать в компании. Доступ можно отключить позже.",
  );
});

test("quickStartForRole returns non-technical steps", () => {
  const steps = quickStartForRole("purchaser");
  assert.ok(steps.length >= 3);
  assert.ok(steps.every((step) => !/api|token|cookie|backend|frontend|company_admin/i.test(step)));
});

test("tourForRole points at on-screen controls without jargon", () => {
  const known = new Set(["order", "north", "jobs", "help", "users", "companies", "company", "company-select"]);
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

test("accessSummaryForRole explains what each role can do", () => {
  assert.match(accessSummaryForRole("platform_admin"), /компани/i);
  assert.match(accessSummaryForRole("company_owner"), /сотрудник/i);
  assert.match(accessSummaryForRole("company_admin"), /приглаш/i);
  assert.match(accessSummaryForRole("purchaser"), /выгрузк/i);
});

test("profileCompanyLabel prefers the company name and keeps platform admin at the service", () => {
  assert.equal(profileCompanyLabel({ role: "platform_admin" }), "Сервис");
  assert.equal(profileCompanyLabel({ role: "purchaser", company_id: "c-1", company_name: "Acme" }), "Acme");
  assert.equal(profileCompanyLabel({ role: "purchaser", company_id: "c-1" }), "Ваша компания");
});

test("headerContext shows company and role for company users", () => {
  assert.deepEqual(headerContext({ role: "company_owner", company_name: "Сияние" }), {
    companyLine: "Компания: Сияние",
    roleLine: "Владелец компании",
  });
});

test("headerContext keeps platform admin at the service", () => {
  assert.deepEqual(headerContext({ role: "platform_admin" }), {
    companyLine: "Сервис",
    roleLine: "Администратор сервиса",
  });
});

test("headerContext falls back when the company name is missing", () => {
  assert.deepEqual(headerContext({ role: "purchaser" }), {
    companyLine: "Компания",
    roleLine: "Закупщик",
  });
});

test("profileFields show login company and access as read-only", () => {
  const fields = profileFields({
    login: "buyer",
    role: "purchaser",
    company_id: "c-1",
    company_name: "Acme",
  });
  assert.deepEqual(
    fields.map((field) => field.label),
    ["Логин", "Компания", "Доступ"],
  );
  assert.equal(fields[0].value, "buyer");
  assert.equal(fields[1].value, "Acme");
  assert.equal(fields[2].value, "Закупщик");
  assert.ok(fields.every((field) => field.editable === false));
});

test("accountPasswordHint tells the user not to share the password", () => {
  assert.equal(
    accountPasswordHint,
    "Пароль только ваш. На работе удобнее входить по Face ID, Touch ID или Windows Hello.",
  );
});

test("logoutEverywhere copy explains that every device will be signed out", () => {
  assert.equal(logoutEverywhereLabel, "Выйти со всех устройств");
  assert.equal(
    logoutEverywhereConfirm,
    "Вы выйдете здесь и на других устройствах. Войти снова можно будет по логину и паролю.",
  );
});

test("two-factor copy is ready for setup and login", () => {
  assert.equal(securitySetupLabel, "Настроить");
  assert.equal(twoFactorEnableLabel, "Включить вход по коду");
  assert.equal(twoFactorOpenAppLabel, "Добавить в приложение");
  assert.equal(twoFactorLoginTitle, "Подтвердите вход");
  assert.equal(twoFactorCodeLabel, "Код из приложения");
  assert.equal(
    twoFactorSetupHint,
    "На работе хватит Face ID или Touch ID. Код нужен, только если входите с чужого компьютера.",
  );
  assert.equal(
    twoFactorRecoveryHint,
    "Сохраните запасные коды: каждый работает один раз.",
  );
  assert.deepEqual(twoFactorSetupSteps, [
    "Откройте Яндекс Ключ, Google Authenticator или 1Password.",
    "На телефоне нажмите «Добавить в приложение» — секрет подставится сам. На компьютере наведите камеру на квадрат.",
    "Введите шесть цифр из приложения.",
  ]);
  assert.equal(
    twoFactorRequiredHint,
    "Добавьте Face ID, Touch ID или Windows Hello. На работе код из приложения не понадобится.",
  );
  assert.equal(
    passkeyInsecureOriginHint,
    "Face ID на этом адресе недоступен. Откройте сайт по обычному домену с https — или войдите паролем.",
  );
});

test("helpSections stay plain and cover the required topics", () => {
  const titles = helpSections.map((section) => section.title);
  assert.deepEqual(titles, [
    "Как сделать выгрузку",
    "Какие файлы нужны",
    'Что значит "Нужно проверить"',
    "Статусы выгрузок",
    "Пользователи и доступ",
    "Если не получается войти",
    "Как быстрее входить",
  ]);
  const text = helpSections.map((section) => `${section.title} ${section.body}`).join("\n");
  assert.equal(/api|token|cookie|backend|frontend|endpoint/i.test(text), false);
  assert.ok(helpSections.every((section) => section.body.split(/(?<=[.!?])\s+/).length <= 2));
});
