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
