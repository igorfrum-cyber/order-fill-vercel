import { formatSessionWhen } from "./session.js";

export const hierarchyBands = [
  { key: "owners", role: "company_owner", title: "Владелец" },
  { key: "admins", role: "company_admin", title: "Администраторы" },
  { key: "purchasers", role: "purchaser", title: "Закупщики" },
];

export function usersByHierarchy(users = []) {
  return hierarchyBands.map((band) => ({
    ...band,
    users: (users || []).filter((user) => user.role === band.role),
  }));
}

export function hierarchyEmptyHint(bandKey) {
  if (bandKey === "purchasers") return "Закупщик заполняет бланки. Пригласите его формой выше.";
  if (bandKey === "admins") return "Администратор приглашает людей и правит профиль компании.";
  return "Владельца назначает администратор сервиса.";
}

export function lastSeenLabel(value) {
  if (!value) return "Ещё не входил";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Ещё не входил";
  const now = new Date();
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) {
    const time = date.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
    return `Вход сегодня, ${time}`;
  }
  const when = formatSessionWhen(value);
  return when ? `Вход ${when}` : "Ещё не входил";
}

export function userInitial(login) {
  return String(login || "?").slice(0, 1).toUpperCase();
}
