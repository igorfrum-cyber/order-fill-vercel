import { roleLabel, accessSummaryForRole, inviteRoleHint } from "../help/copy.js";

export { roleLabel, inviteRoleHint };

export function accessSummary(role) {
  return accessSummaryForRole(role);
}

export function inviteRoleOptions(actorRole) {
  if (actorRole === "platform_admin") {
    return ["company_owner", "company_admin", "purchaser"];
  }
  if (actorRole === "company_owner") {
    return ["company_admin", "purchaser"];
  }
  if (actorRole === "company_admin") {
    return ["purchaser"];
  }
  return [];
}

export function canInviteRole(actorRole, targetRole) {
  return inviteRoleOptions(actorRole).includes(targetRole);
}

export function canEditCompanyProfile(role) {
  return role === "company_owner" || role === "company_admin";
}

export function needsSecurityNudge(me) {
  if (!me || me.two_factor_enabled || me.has_passkey) return false;
  return true;
}

export function canManageListedUser(actorRole, targetRole) {
  if (actorRole === "platform_admin") return true;
  if (actorRole === "company_owner") {
    return targetRole === "company_owner" || targetRole === "company_admin" || targetRole === "purchaser";
  }
  if (actorRole === "company_admin") {
    return targetRole === "company_admin" || targetRole === "purchaser";
  }
  return false;
}

export function needsUsersCompanyPicker(role) {
  return role === "platform_admin";
}

export function homeScreen(role) {
  return role === "platform_admin" ? "overview" : "history";
}

export function resolveUsersCompanyId(role, selectedCompanyId, actorCompanyId) {
  return needsUsersCompanyPicker(role) ? selectedCompanyId || "" : actorCompanyId || "";
}

export function pickDefaultCompanyId(selectedId, companies = []) {
  if (selectedId) return selectedId;
  return companies.find((company) => !company.disabled_at)?.id || "";
}

export function usersCompanyPrompt(companyId, companies = []) {
  if (companyId) return "";
  if (!companies.some((company) => !company.disabled_at)) {
    return "Сначала создайте компанию.";
  }
  return "Выберите компанию, чтобы увидеть сотрудников.";
}

export function companySlugFromPath(pathname) {
  const match = pathname.match(/^\/c\/([^/]+)/);
  return match ? decodeURIComponent(match[1]) : "";
}

export function companyLoginCopy(company) {
  if (!company?.name) {
    return { title: "Вход", lead: "" };
  }
  return {
    title: `Вход для сотрудников «${company.name}»`,
    lead: "Работайте только с файлами своей компании.",
  };
}

export function companyLoginLogoURL(slug) {
  const normalized = normalizeLoginSlug(slug);
  if (loginSlugIssue(normalized)) return "";
  return `/api/v1/public/companies/${encodeURIComponent(normalized)}/logo`;
}

const RESERVED_LOGIN_SLUGS = new Set([
  "admin",
  "api",
  "app",
  "assets",
  "c",
  "ftp",
  "healthz",
  "invite",
  "localhost",
  "login",
  "mail",
  "metrics",
  "public",
  "static",
  "www",
]);

const LOGIN_SLUG_PATTERN = /^[a-z][a-z0-9-]{0,61}[a-z0-9]$/;

export function normalizeLoginSlug(raw) {
  return String(raw || "")
    .trim()
    .toLowerCase();
}

export function loginSlugIssue(raw) {
  const slug = normalizeLoginSlug(raw);
  if (!slug) return "Укажите адрес входа латиницей.";
  if (RESERVED_LOGIN_SLUGS.has(slug)) return "Этот адрес зарезервирован.";
  if (!LOGIN_SLUG_PATTERN.test(slug)) return "Только латиница, цифры и дефис. Без пробелов и кириллицы.";
  return "";
}

export function companyLoginPath(slug) {
  return `/c/${encodeURIComponent(normalizeLoginSlug(slug))}`;
}

export function companySlugFromHost(hostname) {
  const host = String(hostname || "")
    .split(":")[0]
    .toLowerCase();
  if (!host || host === "localhost" || isIPAddress(host)) return "";
  const labels = host.split(".");
  if (labels.length < 2) return "";
  const slug = labels[0];
  if (loginSlugIssue(slug)) return "";
  return slug;
}

export function loginParentHost(hostname) {
  const host = String(hostname || "")
    .split(":")[0]
    .toLowerCase();
  if (!host || isIPAddress(host) || host === "localhost" || host.endsWith(".localhost")) {
    return "localhost";
  }
  const labels = host.split(".");
  if (labels[0] === "www") {
    return labels.slice(1).join(".") || host;
  }
  if (labels.length >= 3 && !loginSlugIssue(labels[0])) {
    return labels.slice(1).join(".");
  }
  return host;
}

export function companyLoginURL(slug, location = globalThis.location) {
  const normalized = normalizeLoginSlug(slug);
  if (loginSlugIssue(normalized)) return "";
  const parent = loginParentHost(location?.hostname || "localhost");
  const protocol = location?.protocol || "http:";
  const port = location?.port || "";
  const portPart = port ? `:${port}` : "";
  return `${protocol}//${normalized}.${parent}${portPart}/`;
}

function isIPAddress(host) {
  return /^\d{1,3}(?:\.\d{1,3}){3}$/.test(host) || host.includes(":");
}
