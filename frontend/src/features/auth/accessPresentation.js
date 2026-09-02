import { roleLabel } from "../help/copy.js";

const INVITE_ROLES = ["purchaser", "company_admin"];

export { roleLabel };

export function accessSummary(role) {
  if (role === "platform_admin") {
    return "Вы можете создавать компании, помогать с доступом и смотреть историю по всем компаниям.";
  }
  if (role === "company_admin") {
    return "Вы приглашаете сотрудников, сбрасываете доступ и видите выгрузки своей компании.";
  }
  return "Вы создаёте выгрузки, проверяете строки и скачиваете готовые файлы.";
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
