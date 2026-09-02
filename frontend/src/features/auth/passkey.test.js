import assert from "node:assert/strict";
import test from "node:test";

import { bytesToBase64url, passkeySupported, toBuffer } from "./passkey.js";

test("passkeySupported is false without WebAuthn", () => {
  const previous = globalThis.PublicKeyCredential;
  const credentials = globalThis.navigator;
  globalThis.PublicKeyCredential = undefined;
  Object.defineProperty(globalThis, "navigator", { configurable: true, value: {} });
  assert.equal(passkeySupported(), false);
  globalThis.PublicKeyCredential = previous;
  Object.defineProperty(globalThis, "navigator", { configurable: true, value: credentials });
});

test("base64url round-trip does not use plus or slash", () => {
  const bytes = new Uint8Array([255, 239, 0, 1]).buffer;
  const encoded = bytesToBase64url(bytes);
  assert.equal(encoded.includes("+"), false);
  assert.equal(encoded.includes("/"), false);
  assert.deepEqual(new Uint8Array(toBuffer(encoded)), new Uint8Array(bytes));
});
