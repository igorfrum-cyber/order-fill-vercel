import { passkeyInsecureOriginHint } from "../help/copy.js";
import { userFacingError } from "../help/errors.js";

export function defaultPasskeyName(existingCount = 0) {
  const count = Number(existingCount) || 0;
  if (count < 1) return "Это устройство";
  return `Это устройство ${count + 1}`;
}

export function passkeySupported() {
  return typeof globalThis.PublicKeyCredential === "function" && typeof navigator.credentials?.create === "function";
}

export function passkeyOriginIssue(location = globalThis.location) {
  const hostname = String(location?.hostname || "")
    .split(":")[0]
    .toLowerCase();
  if (!hostname || isLoopbackPasskeyHost(hostname)) return "";
  if (isIPAddress(hostname)) return "insecure";
  if (String(location?.protocol || "") === "https:") return "";
  return "insecure";
}

export function passkeyUsable(location = globalThis.location) {
  return passkeySupported() && !passkeyOriginIssue(location);
}

function isLoopbackPasskeyHost(host) {
  return host === "localhost" || host === "127.0.0.1" || host === "::1" || host.endsWith(".localhost");
}

function isIPAddress(host) {
  return /^\d{1,3}(?:\.\d{1,3}){3}$/.test(host) || (host.includes(":") && host !== "::1");
}

export async function conditionalMediationAvailable() {
  if (!passkeySupported() || typeof PublicKeyCredential.isConditionalMediationAvailable !== "function") {
    return false;
  }
  try {
    return Boolean(await PublicKeyCredential.isConditionalMediationAvailable());
  } catch {
    return false;
  }
}

export function creationOptionsFromJSON(publicKey) {
  return decodePublicKeyOptions(publicKey, true);
}

export function requestOptionsFromJSON(publicKey) {
  return decodePublicKeyOptions(publicKey, false);
}

export function isPasskeyRequestPending(err) {
  return /already pending|request is pending|operation is pending/i.test(String(err?.message || ""));
}

export async function waitForPasskeySlot() {
  await new Promise((resolve) => setTimeout(resolve, 100));
}

export function passkeyErrorMessage(err, action = "add") {
  const name = err?.name || "";
  if (isPasskeyRequestPending(err)) {
    return action === "login"
      ? "Ключ доступа ещё занят. Нажмите «Войти по Face ID или Touch ID» ещё раз."
      : "Ключ доступа ещё занят. Нажмите ещё раз.";
  }
  if (name === "NotAllowedError" || name === "AbortError") {
    return action === "login" ? "Вход с ключом доступа отменён." : "Добавление отменено.";
  }
  if (name === "SecurityError") {
    if (passkeyOriginIssue()) return passkeyInsecureOriginHint;
    return "Этот адрес страницы не подходит для ключа доступа. Откройте страницу компании и попробуйте снова.";
  }
  if (name === "InvalidStateError") {
    return "Этот ключ доступа уже добавлен.";
  }
  if (name === "NotSupportedError") {
    if (passkeyOriginIssue()) return passkeyInsecureOriginHint;
    return "Этот браузер или приложение не умеет создавать ключ доступа.";
  }
  if (name === "ConstraintError") {
    return "Это устройство не подходит. Выберите другое приложение или ключ.";
  }
  if (name === "TypeError" || name === "DataError" || name === "EncodingError" || name === "SyntaxError") {
    return "Параметры ключа доступа с сервера неверные. Обновите страницу и попробуйте снова.";
  }
  if (name === "ApiError") {
    return userFacingError(err, action === "login" ? "Не получилось войти с ключом доступа. Можно войти паролем." : "Не удалось добавить ключ доступа. Попробуйте другое приложение или ключ.");
  }
  if (typeof err?.message === "string" && err.message.trim()) {
    return userFacingError(err, action === "login" ? "Не получилось войти с ключом доступа. Можно войти паролем." : "Не удалось добавить ключ доступа. Попробуйте другое приложение или ключ.");
  }
  return action === "login"
    ? "Не получилось войти с ключом доступа. Можно войти паролем."
    : "Не удалось добавить ключ доступа. Попробуйте другое приложение или ключ.";
}

export function credentialToJSON(credential) {
  if (typeof credential?.toJSON === "function") {
    return credential.toJSON();
  }
  return encodeCredential(credential);
}

export function publicKeyFromBegin(options) {
  if (!options) return null;
  return options.publicKey || options;
}

function decodePublicKeyOptions(publicKey, creation) {
  if (!publicKey || typeof publicKey !== "object") {
    throw new TypeError("сервер не прислал параметры ключа доступа");
  }
  if (!publicKey.challenge) {
    throw new TypeError("нет challenge в параметрах ключа доступа");
  }
  const next = { ...publicKey, challenge: toBuffer(publicKey.challenge) };
  if (creation) {
    if (!publicKey.user?.id) {
      throw new TypeError("нет user.id в параметрах ключа доступа");
    }
    next.user = { ...publicKey.user, id: toBuffer(publicKey.user.id) };
  }
  const listKey = creation ? "excludeCredentials" : "allowCredentials";
  if (Array.isArray(publicKey[listKey])) {
    next[listKey] = publicKey[listKey].map((item) => ({ ...item, id: toBuffer(item.id) }));
  }
  return next;
}

function encodeCredential(credential) {
  const response = credential.response;
  const payload = {
    id: credential.id,
    rawId: bytesToBase64url(credential.rawId),
    type: credential.type || "public-key",
    authenticatorAttachment: credential.authenticatorAttachment || undefined,
    response: {},
    clientExtensionResults: credential.getClientExtensionResults?.() || {},
  };
  if (response.attestationObject) {
    payload.response.clientDataJSON = bytesToBase64url(response.clientDataJSON);
    payload.response.attestationObject = bytesToBase64url(response.attestationObject);
    if (response.getTransports) payload.response.transports = response.getTransports();
  } else {
    payload.response.clientDataJSON = bytesToBase64url(response.clientDataJSON);
    payload.response.authenticatorData = bytesToBase64url(response.authenticatorData);
    payload.response.signature = bytesToBase64url(response.signature);
    if (response.userHandle) payload.response.userHandle = bytesToBase64url(response.userHandle);
  }
  return payload;
}

export function bytesToBase64url(buffer) {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/g, "");
}

export function toBuffer(value) {
  if (value instanceof ArrayBuffer) return value;
  if (ArrayBuffer.isView(value)) return value.buffer;
  if (typeof value !== "string") {
    throw new Error("unsupported passkey value");
  }
  const padded = value.replaceAll("-", "+").replaceAll("_", "/");
  const pad = padded.length % 4 === 0 ? "" : "=".repeat(4 - (padded.length % 4));
  const binary = atob(padded + pad);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}
