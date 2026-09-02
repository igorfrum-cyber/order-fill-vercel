export function isAccessAudit(action) {
  return ["password_changed", "invite_created", "access_reset", "user_disabled", "company_disabled"].includes(action);
}

export function auditLine(event = {}) {
  const who = event.actor_login || "Кто-то";
  const company = event.company_name ? `«${event.company_name}»` : "";
  switch (event.action) {
    case "invite_created":
      return company ? `${who} пригласил человека в ${company}` : `${who} отправил приглашение`;
    case "access_reset":
      return company ? `${who} сбросил доступ в ${company}` : `${who} сбросил доступ`;
    case "user_disabled":
      return company ? `${who} отключил сотрудника в ${company}` : `${who} отключил сотрудника`;
    case "company_disabled":
      return company ? `${who} отключил компанию ${company}` : `${who} отключил компанию`;
    case "password_changed":
      return `${who} сменил пароль`;
    default:
      return "";
  }
}

export function accessAuditEvents(events = []) {
  return (events || []).filter((event) => isAccessAudit(event.action)).map((event) => ({
    ...event,
    line: auditLine(event),
  }));
}
