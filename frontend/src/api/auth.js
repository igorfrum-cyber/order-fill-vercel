import { apiClient } from "./client.js";

export function getMe() {
  return apiClient.request("/api/v1/auth/me");
}

export function getCompanyLogin(slug) {
  return apiClient.request(`/api/v1/public/companies/${encodeURIComponent(slug)}/login`);
}

export function login(login, password) {
  return apiClient.request("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ login, password }),
  });
}

export function completeTwoFactorLogin(challengeId, code) {
  return apiClient.request("/api/v1/auth/login/2fa", {
    method: "POST",
    body: JSON.stringify({ challenge_id: challengeId, code }),
  });
}

export function startTwoFactorSetup() {
  return apiClient.request("/api/v1/auth/2fa/setup", {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export function enableTwoFactor(code) {
  return apiClient.request("/api/v1/auth/2fa/enable", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
}

export function disableTwoFactor(password) {
  return apiClient.request("/api/v1/auth/2fa/disable", {
    method: "POST",
    body: JSON.stringify({ password }),
  });
}

export function acceptInvite(token, password) {
  return apiClient.request("/api/v1/auth/invite", {
    method: "POST",
    body: JSON.stringify({ token, password }),
  });
}

export function logout() {
  return apiClient.request("/api/v1/auth/logout", { method: "POST" });
}

export function logoutEverywhere() {
  return apiClient.request("/api/v1/auth/logout-everywhere", { method: "POST" });
}

export function changePassword(currentPassword, password) {
  return apiClient.request("/api/v1/auth/password", {
    method: "POST",
    body: JSON.stringify({ current_password: currentPassword, password }),
  });
}

export function beginPasskeyRegistration(name = "") {
  return apiClient.request("/api/v1/auth/passkeys/register/begin", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export function finishPasskeyRegistration(challengeId, credential, name = "") {
  return apiClient.request("/api/v1/auth/passkeys/register/finish", {
    method: "POST",
    body: JSON.stringify({ challenge_id: challengeId, credential, name }),
  });
}

export function listPasskeys() {
  return apiClient.request("/api/v1/auth/passkeys");
}

export function listSessions() {
  return apiClient.request("/api/v1/auth/sessions");
}

export function revokeSession(id) {
  return apiClient.request(`/api/v1/auth/sessions/${encodeURIComponent(id)}/delete`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export function deletePasskey(id) {
  return apiClient.request(`/api/v1/auth/passkeys/${encodeURIComponent(id)}/delete`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export function beginPasskeyLogin(login = "") {
  return apiClient.request("/api/v1/auth/passkeys/login/begin", {
    method: "POST",
    body: JSON.stringify({ login }),
  });
}

export function finishPasskeyLogin(challengeId, credential) {
  return apiClient.request("/api/v1/auth/passkeys/login/finish", {
    method: "POST",
    body: JSON.stringify({ challenge_id: challengeId, credential }),
  });
}

export function listJobs(companyId = "") {
  const query = companyId ? `?company_id=${encodeURIComponent(companyId)}` : "";
  return apiClient.request(`/api/v1/jobs${query}`);
}

export function listCompanies() {
  return apiClient.request("/api/v1/companies");
}

export function createCompany(name, loginSlug) {
  return apiClient.request("/api/v1/companies", {
    method: "POST",
    body: JSON.stringify({ name, login_slug: loginSlug }),
  });
}

export function setCompanyLoginSlug(companyId, loginSlug) {
  return apiClient.request(`/api/v1/companies/${encodeURIComponent(companyId)}/login-slug`, {
    method: "POST",
    body: JSON.stringify({ login_slug: loginSlug }),
  });
}

export function updateCompany(companyId, name, loginSlug) {
  return apiClient.request(`/api/v1/companies/${encodeURIComponent(companyId)}/profile`, {
    method: "POST",
    body: JSON.stringify({ name, login_slug: loginSlug }),
  });
}

export function setCompanyLogo(companyId, file) {
  const body = new FormData();
  body.append("logo", file);
  return apiClient.request(`/api/v1/companies/${encodeURIComponent(companyId)}/logo`, {
    method: "POST",
    body,
  });
}

export function clearCompanyLogo(companyId) {
  return apiClient.request(`/api/v1/companies/${encodeURIComponent(companyId)}/logo/clear`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export function disableCompany(companyId) {
  return apiClient.request(`/api/v1/companies/${encodeURIComponent(companyId)}/disable`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export function listUsers(companyId) {
  return apiClient.request(`/api/v1/companies/${encodeURIComponent(companyId)}/users`);
}

export function createUser(companyId, login, role) {
  return apiClient.request(`/api/v1/companies/${encodeURIComponent(companyId)}/users`, {
    method: "POST",
    body: JSON.stringify({ login, role }),
  });
}

export function disableUser(userId) {
  return apiClient.request(`/api/v1/users/${encodeURIComponent(userId)}/disable`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export function resetUser(userId) {
  return apiClient.request(`/api/v1/users/${encodeURIComponent(userId)}/reset`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export function listAudit() {
  return apiClient.request("/api/v1/audit");
}

export function listStatus() {
  return apiClient.request("/api/v1/status");
}
