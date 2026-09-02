import assert from "node:assert/strict";
import test from "node:test";

import { generatePassword, isPasswordReady, passwordIssues } from "./password.js";

test("generatePassword is long enough and mixes letter with digit", () => {
  const password = generatePassword();
  assert.equal(password.length, 16);
  assert.equal(passwordIssues(password, password).length, 0);
  assert.equal(isPasswordReady(password, password), true);
});

test("passwordIssues flags short, spaced and mismatched values", () => {
  assert.deepEqual(passwordIssues("short", "short"), ["минимум 10 символов", "хотя бы одна цифра"]);
  assert.ok(passwordIssues("abcdefghij1", "abcdefghij1 ").includes("пароли не совпадают"));
  assert.ok(passwordIssues("password 12", "password 12").includes("без пробелов"));
});
