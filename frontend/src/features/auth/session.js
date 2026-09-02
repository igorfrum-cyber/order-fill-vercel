export function sessionIsPhone(device) {
  return /iPhone|iPad|Android/i.test(String(device || ""));
}

export function formatSessionWhen(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString("ru-RU", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
}

export function passkeyWhen(item = {}) {
  if (item.last_used_at) {
    const when = formatSessionWhen(item.last_used_at);
    return when ? `последний вход ${when}` : "";
  }
  if (item.created_at) {
    const when = formatSessionWhen(item.created_at);
    return when ? `добавлен ${when}` : "";
  }
  return "";
}
