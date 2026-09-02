import assert from "node:assert/strict";
import test from "node:test";

import { twoFactorOpenAppHref } from "./twoFactor.js";

test("twoFactorOpenAppHref only allows otpauth links from setup", () => {
  assert.equal(
    twoFactorOpenAppHref("otpauth://totp/Order%20Fill:buyer?secret=ABC&issuer=Order%20Fill"),
    "otpauth://totp/Order%20Fill:buyer?secret=ABC&issuer=Order%20Fill",
  );
  assert.equal(twoFactorOpenAppHref("  otpauth://totp/x?secret=1  "), "otpauth://totp/x?secret=1");
  assert.equal(twoFactorOpenAppHref("https://evil.example/otpauth://totp/x"), "");
  assert.equal(twoFactorOpenAppHref("javascript:alert(1)"), "");
  assert.equal(twoFactorOpenAppHref(""), "");
});
