import { roleLabel, accessSummaryForRole } from "../help/copy.js";

const INVITE_ROLES = ["purchaser", "company_admin"];

export { roleLabel };

export function accessSummary(role) {
  return accessSummaryForRole(role);
}

export function inviteRoleOptions(actorRole) {
  if (actorRole === "platform_admin") {
    return ["company_admin"];
  }
  if (actorRole === "company_admin") {
    return [...INVITE_ROLES];
  }
  return [];
}

export function canInviteRole(actorRole, targetRole) {
  return inviteRoleOptions(actorRole).includes(targetRole);
}

export function needsUsersCompanyPicker(role) {
  return role === "platform_admin";
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
