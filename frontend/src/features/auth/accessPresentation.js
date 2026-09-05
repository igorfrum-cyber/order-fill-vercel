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
  if (role === "platform_admin") return "overview";
  if (role === "purchaser") return "order";
  return "history";
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
  if (!host || isLoopbackHost(host)) {
    return "localhost";
  }
  const registrable = registrableHost(host);
  if (registrable) return registrable;
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
  const hostname = String(location?.hostname || "localhost")
    .split(":")[0]
    .toLowerCase();
  const protocol = location?.protocol || "http:";
  const port = location?.port || "";
  const portPart = port ? `:${port}` : "";
  if (usesPathCompanyLogin(hostname)) {
    return `${protocol}//${hostForURL(hostname)}${portPart}${companyLoginPath(normalized)}`;
  }
  const parent = loginParentHost(hostname);
  return `${protocol}//${normalized}.${parent}${portPart}/`;
}

const PATH_LOGIN_SUFFIXES = ["duckdns.org", "sslip.io", "nip.io"];

function usesPathCompanyLogin(host) {
  if (isIPAddress(host) && !isLoopbackHost(host)) return true;
  return PATH_LOGIN_SUFFIXES.some((suffix) => host === suffix || host.endsWith(`.${suffix}`));
}

function registrableHost(host) {
  for (const suffix of PATH_LOGIN_SUFFIXES) {
    if (host === suffix) return suffix;
    if (!host.endsWith(`.${suffix}`)) continue;
    const rest = host.slice(0, -(suffix.length + 1));
    const head = rest.split(".");
    return `${head[head.length - 1]}.${suffix}`;
  }
  return "";
}

function isLoopbackHost(host) {
  return host === "localhost" || host === "127.0.0.1" || host === "::1" || host.endsWith(".localhost");
}

function isIPAddress(host) {
  return /^\d{1,3}(?:\.\d{1,3}){3}$/.test(host) || host.includes(":");
}

function hostForURL(host) {
  if (host.includes(":") && !host.startsWith("[")) return `[${host}]`;
  return host;
}
