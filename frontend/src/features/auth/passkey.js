export function passkeySupported() {
  return typeof globalThis.PublicKeyCredential === "function" && typeof navigator.credentials?.create === "function";
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
  if (typeof PublicKeyCredential.parseCreationOptionsFromJSON === "function") {
    return PublicKeyCredential.parseCreationOptionsFromJSON(publicKey);
  }
  return decodePublicKeyOptions(publicKey, true);
}

export function requestOptionsFromJSON(publicKey) {
  if (typeof PublicKeyCredential.parseRequestOptionsFromJSON === "function") {
    return PublicKeyCredential.parseRequestOptionsFromJSON(publicKey);
  }
  return decodePublicKeyOptions(publicKey, false);
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
  const next = { ...publicKey, challenge: toBuffer(publicKey.challenge) };
  if (creation && publicKey.user) {
    next.user = { ...publicKey.user, id: decodeUserID(publicKey.user.id) };
  }
  const listKey = creation ? "excludeCredentials" : "allowCredentials";
  if (Array.isArray(publicKey[listKey])) {
    next[listKey] = publicKey[listKey].map((item) => ({ ...item, id: toBuffer(item.id) }));
  }
  return next;
}

function decodeUserID(value) {
  if (typeof value === "string" && !looksLikeBase64url(value)) {
    return new TextEncoder().encode(value).buffer;
  }
  return toBuffer(value);
}

function looksLikeBase64url(value) {
  return /^[A-Za-z0-9_-]+$/.test(value) && value.length % 4 !== 1;
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
