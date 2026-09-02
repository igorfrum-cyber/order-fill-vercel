import assert from "node:assert/strict";
import test from "node:test";

import {
  bytesToBase64url,
  creationOptionsFromJSON,
  defaultPasskeyName,
  passkeyErrorMessage,
  isPasskeyRequestPending,
  passkeyOriginIssue,
  passkeySupported,
  passkeyUsable,
} from "./passkey.js";

test("defaultPasskeyName names the current device without asking", () => {
  assert.equal(defaultPasskeyName(0), "Это устройство");
  assert.equal(defaultPasskeyName(1), "Это устройство 2");
});

test("passkeyOriginIssue allows localhost and https domains", () => {
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "127.0.0.1" }), "");
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "localhost" }), "");
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "kristail.localhost" }), "");
  assert.equal(passkeyOriginIssue({ protocol: "https:", hostname: "kristail.example.com" }), "");
  assert.equal(passkeyOriginIssue({ protocol: "https:", hostname: "example.com" }), "");
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "192.168.31.108" }), "insecure");
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "example.com" }), "insecure");
});

test("passkeyUsable is false on a LAN IP even if WebAuthn exists", () => {
  const previous = globalThis.PublicKeyCredential;
  function FakePublicKeyCredential() {}
  globalThis.PublicKeyCredential = FakePublicKeyCredential;
  const credentials = globalThis.navigator;
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: { credentials: { create: async () => null } },
  });
  try {
    assert.equal(passkeySupported(), true);
    assert.equal(passkeyUsable({ protocol: "http:", hostname: "192.168.31.108" }), false);
    assert.equal(passkeyUsable({ protocol: "https:", hostname: "kristail.example.com" }), true);
  } finally {
    globalThis.PublicKeyCredential = previous;
    Object.defineProperty(globalThis, "navigator", { configurable: true, value: credentials });
  }
});

test("passkeySupported is false without WebAuthn", () => {
  const previous = globalThis.PublicKeyCredential;
  const credentials = globalThis.navigator;
  globalThis.PublicKeyCredential = undefined;
  Object.defineProperty(globalThis, "navigator", { configurable: true, value: {} });
  assert.equal(passkeySupported(), false);
  globalThis.PublicKeyCredential = previous;
  Object.defineProperty(globalThis, "navigator", { configurable: true, value: credentials });
});

test("creationOptionsFromJSON always decodes itself without the browser helper", () => {
  const previous = globalThis.PublicKeyCredential;
  function FakePublicKeyCredential() {}
  FakePublicKeyCredential.parseCreationOptionsFromJSON = () => {
    throw new Error("browser helper must not be used");
  };
  globalThis.PublicKeyCredential = FakePublicKeyCredential;
  try {
    const userID = bytesToBase64url(new TextEncoder().encode("user-1"));
    const options = creationOptionsFromJSON({
      rp: { id: "christyle.localhost", name: "Order Fill" },
      user: { id: userID, name: "buyer", displayName: "buyer" },
      challenge: bytesToBase64url(new Uint8Array([1, 2, 3, 4]).buffer),
      pubKeyCredParams: [{ type: "public-key", alg: -7 }],
    });
    assert.equal(options.rp.id, "christyle.localhost");
    assert.ok(options.challenge instanceof ArrayBuffer);
    assert.equal(new TextDecoder().decode(options.user.id), "user-1");
    assert.deepEqual(new Uint8Array(options.challenge), new Uint8Array([1, 2, 3, 4]));
  } finally {
    globalThis.PublicKeyCredential = previous;
  }
});

test("passkeyErrorMessage explains browser security errors", () => {
  const security = new Error("The relying party ID is not a registrable domain suffix");
  security.name = "SecurityError";
  assert.equal(
    passkeyErrorMessage(security, "add"),
    "Этот адрес страницы не подходит для ключа доступа. Откройте страницу компании и попробуйте снова.",
  );
  const cancelled = new Error("cancelled");
  cancelled.name = "NotAllowedError";
  assert.equal(passkeyErrorMessage(cancelled, "add"), "Добавление отменено.");
  const duplicate = new Error("already registered");
  duplicate.name = "InvalidStateError";
  assert.equal(passkeyErrorMessage(duplicate, "add"), "Этот ключ доступа уже добавлен.");
  const api = new Error("challenge expired");
  api.name = "ApiError";
  assert.equal(passkeyErrorMessage(api, "add"), "challenge expired");
  const pending = new Error("A request is already pending.");
  pending.name = "NotAllowedError";
  assert.equal(
    passkeyErrorMessage(pending, "login"),
    "Ключ доступа ещё занят. Нажмите «Войти по Face ID или Touch ID» ещё раз.",
  );
  assert.equal(isPasskeyRequestPending(pending), true);
});
