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
  if (actorRole === "platform_admin" || actorRole === "company_admin") {
    return [...INVITE_ROLES];
  }
  return [];
}

export function canInviteRole(actorRole, targetRole) {
  return inviteRoleOptions(actorRole).includes(targetRole);
}
