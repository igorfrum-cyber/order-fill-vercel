import assert from "node:assert/strict";
import test from "node:test";

import { loginAccessHint, loginFailedMessage } from "./copy.js";

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
