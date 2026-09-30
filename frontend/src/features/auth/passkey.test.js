import assert from "node:assert/strict";
import test from "node:test";

import {
  bytesToBase64url,
  creationOptionsFromJSON,
  credentialToJSON,
  conditionalMediationAvailable,
  defaultPasskeyName,
  isPasskeyRequestPending,
  passkeyErrorMessage,
  passkeyOriginIssue,
  passkeySupported,
  passkeyUsable,
  publicKeyFromBegin,
  requestOptionsFromJSON,
  toBuffer,
  waitForPasskeySlot,
} from "./passkey.js";

test("defaultPasskeyName names the current device without asking", () => {
  assert.equal(defaultPasskeyName(0), "Это устройство");
  assert.equal(defaultPasskeyName(1), "Это устройство 2");
});

test("passkeyOriginIssue allows localhost and https domains", () => {
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "127.0.0.1" }), "insecure");
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
    assert.equal(passkeyUsable({ protocol: "http:", hostname: "127.0.0.1" }), false);
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
  assert.equal(passkeyErrorMessage(api, "add"), "Код или ссылка устарели. Попробуйте ещё раз.");
  const pending = new Error("A request is already pending.");
  pending.name = "NotAllowedError";
  assert.equal(
    passkeyErrorMessage(pending, "login"),
    "Ключ доступа ещё занят. Нажмите «Войти по Face ID или Touch ID» ещё раз.",
  );
  assert.equal(isPasskeyRequestPending(pending), true);
});

test("passkey helpers normalize names, origins and browser support edge cases", () => {
  assert.equal(defaultPasskeyName(-2), "Это устройство");
  assert.equal(defaultPasskeyName("2"), "Это устройство 3");
  assert.equal(passkeyOriginIssue(null), "");
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "[::1]" }), "insecure");
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "::1" }), "");
  assert.equal(passkeyOriginIssue({ protocol: "http:", hostname: "laptop.localhost" }), "");
  assert.equal(isPasskeyRequestPending(new Error("not pending")), false);
  assert.equal(isPasskeyRequestPending(null), false);
  assert.deepEqual(publicKeyFromBegin(null), null);
  assert.deepEqual(publicKeyFromBegin({ challenge: "c" }), { challenge: "c" });
  assert.deepEqual(publicKeyFromBegin({ publicKey: { challenge: "c" }, id: "outer" }), { challenge: "c" });

  const previousCredential = globalThis.PublicKeyCredential;
  const previousNavigator = globalThis.navigator;
  try {
    globalThis.PublicKeyCredential = function PublicKeyCredential() {};
    Object.defineProperty(globalThis, "navigator", { configurable: true, value: { credentials: {} } });
    assert.equal(passkeySupported(), false);
    assert.equal(passkeyUsable({ protocol: "https:", hostname: "example.com" }), false);
    Object.defineProperty(globalThis, "navigator", { configurable: true, value: { credentials: { create() {} } } });
    assert.equal(passkeySupported(), true);
  } finally {
    globalThis.PublicKeyCredential = previousCredential;
    Object.defineProperty(globalThis, "navigator", { configurable: true, value: previousNavigator });
  }
});

test("conditional mediation availability handles unsupported, success, false and rejection", async () => {
  const previousCredential = globalThis.PublicKeyCredential;
  const previousNavigator = globalThis.navigator;
  try {
    globalThis.PublicKeyCredential = undefined;
    Object.defineProperty(globalThis, "navigator", { configurable: true, value: {} });
    assert.equal(await conditionalMediationAvailable(), false);

    function FakePublicKeyCredential() {}
    globalThis.PublicKeyCredential = FakePublicKeyCredential;
    Object.defineProperty(globalThis, "navigator", { configurable: true, value: { credentials: { create() {} } } });
    assert.equal(await conditionalMediationAvailable(), false);
    FakePublicKeyCredential.isConditionalMediationAvailable = async () => true;
    assert.equal(await conditionalMediationAvailable(), true);
    FakePublicKeyCredential.isConditionalMediationAvailable = async () => false;
    assert.equal(await conditionalMediationAvailable(), false);
    FakePublicKeyCredential.isConditionalMediationAvailable = async () => { throw new Error("unsupported"); };
    assert.equal(await conditionalMediationAvailable(), false);
  } finally {
    globalThis.PublicKeyCredential = previousCredential;
    Object.defineProperty(globalThis, "navigator", { configurable: true, value: previousNavigator });
  }
});

test("registration and assertion options decode challenge and credential IDs", () => {
  const encoded = bytesToBase64url(new Uint8Array([1, 2, 3]).buffer);
  const registration = creationOptionsFromJSON({
    challenge: encoded,
    user: { id: encoded, name: "buyer" },
    excludeCredentials: [{ id: encoded, type: "public-key" }],
  });
  assert.deepEqual([...new Uint8Array(registration.challenge)], [1, 2, 3]);
  assert.deepEqual([...new Uint8Array(registration.user.id)], [1, 2, 3]);
  assert.deepEqual([...new Uint8Array(registration.excludeCredentials[0].id)], [1, 2, 3]);

  const assertion = requestOptionsFromJSON({
    challenge: encoded,
    allowCredentials: [{ id: encoded, transports: ["internal"] }],
  });
  assert.deepEqual([...new Uint8Array(assertion.allowCredentials[0].id)], [1, 2, 3]);
  assert.equal(requestOptionsFromJSON({ challenge: encoded }).challenge instanceof ArrayBuffer, true);

  assert.throws(() => creationOptionsFromJSON(null), /сервер не прислал параметры/);
  assert.throws(() => requestOptionsFromJSON("invalid"), /сервер не прислал параметры/);
  assert.throws(() => requestOptionsFromJSON({}), /нет challenge/);
  assert.throws(() => creationOptionsFromJSON({ challenge: encoded }), /нет user.id/);
  assert.throws(() => toBuffer(42), /unsupported passkey value/);
  const view = new Uint8Array([4, 5]);
  assert.equal(toBuffer(view), view.buffer);
});

test("credential serialization handles native JSON and registration and assertion responses", () => {
  assert.deepEqual(credentialToJSON({ toJSON: () => ({ id: "native" }) }), { id: "native" });

  const registration = credentialToJSON({
    id: "credential-1",
    rawId: new Uint8Array([1, 2]).buffer,
    response: {
      clientDataJSON: new Uint8Array([3]).buffer,
      attestationObject: new Uint8Array([4]).buffer,
      getTransports: () => ["internal"],
    },
    getClientExtensionResults: () => ({ appid: true }),
  });
  assert.equal(registration.rawId, "AQI");
  assert.equal(registration.response.clientDataJSON, "Aw");
  assert.equal(registration.response.attestationObject, "BA");
  assert.deepEqual(registration.response.transports, ["internal"]);
  assert.deepEqual(registration.clientExtensionResults, { appid: true });

  const assertion = credentialToJSON({
    id: "credential-2",
    rawId: new Uint8Array([5]).buffer,
    type: "",
    response: {
      clientDataJSON: new Uint8Array([6]).buffer,
      authenticatorData: new Uint8Array([7]).buffer,
      signature: new Uint8Array([8]).buffer,
      userHandle: new Uint8Array([9]).buffer,
    },
  });
  assert.equal(assertion.type, "public-key");
  assert.equal(assertion.response.authenticatorData, "Bw");
  assert.equal(assertion.response.signature, "CA");
  assert.equal(assertion.response.userHandle, "CQ");
  assert.deepEqual(assertion.clientExtensionResults, {});
  assert.equal(bytesToBase64url(new Uint8Array([255, 254]).buffer), "__4");
});

test("passkey error messages cover cancellation, unsupported browser, server data and fallback", () => {
  for (const name of ["AbortError", "NotAllowedError"]) {
    const error = Object.assign(new Error("cancelled"), { name });
    assert.equal(passkeyErrorMessage(error, "login"), "Вход с ключом доступа отменён.");
  }
  assert.equal(passkeyErrorMessage(Object.assign(new Error(), { name: "NotSupportedError" })), "Этот браузер или приложение не умеет создавать ключ доступа.");
  assert.equal(passkeyErrorMessage(Object.assign(new Error(), { name: "ConstraintError" })), "Это устройство не подходит. Выберите другое приложение или ключ.");
  for (const name of ["TypeError", "DataError", "EncodingError", "SyntaxError"]) {
    assert.match(passkeyErrorMessage(Object.assign(new Error(), { name })), /Параметры ключа доступа/);
  }
  assert.equal(passkeyErrorMessage(Object.assign(new Error("bad response"), { name: "ApiError", status: 500 }), "login"), "Не получилось войти с ключом доступа. Можно войти паролем.");
  assert.equal(passkeyErrorMessage(new Error("  ")), "Не удалось добавить ключ доступа. Попробуйте другое приложение или ключ.");
  assert.equal(passkeyErrorMessage(null, "login"), "Не получилось войти с ключом доступа. Можно войти паролем.");
  assert.equal(typeof passkeyErrorMessage(new Error("unexpected passkey error")), "string");
});

test("waitForPasskeySlot yields through a short timer", async () => {
  const originalSetTimeout = globalThis.setTimeout;
  let delay;
  globalThis.setTimeout = (callback, milliseconds) => {
    delay = milliseconds;
    callback();
    return 0;
  };
  try {
    await waitForPasskeySlot();
    assert.equal(delay, 100);
  } finally {
    globalThis.setTimeout = originalSetTimeout;
  }
});
