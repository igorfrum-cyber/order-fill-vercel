import {
  beginPasskeyLogin,
  beginPasskeyRegistration,
  finishPasskeyLogin,
  finishPasskeyRegistration,
} from "../../api/auth.js";
import {
  creationOptionsFromJSON,
  credentialToJSON,
  publicKeyFromBegin,
  requestOptionsFromJSON,
} from "./passkey.js";

export async function createPasskey(name = "") {
  const begin = await beginPasskeyRegistration(name);
  const publicKey = creationOptionsFromJSON(publicKeyFromBegin(begin.options));
  const credential = await navigator.credentials.create({ publicKey });
  if (!credential) {
    const cancelled = new Error("Добавление отменено.");
    cancelled.name = "NotAllowedError";
    throw cancelled;
  }
  return finishPasskeyRegistration(begin.challenge_id, credentialToJSON(credential), name);
}

export async function authenticatePasskey(login = "", { mediation, signal } = {}) {
  const begin = await beginPasskeyLogin(login);
  const publicKey = requestOptionsFromJSON(publicKeyFromBegin(begin.options));
  const credential = await navigator.credentials.get({ publicKey, mediation, signal });
  if (!credential) return null;
  return finishPasskeyLogin(begin.challenge_id, credentialToJSON(credential));
}
