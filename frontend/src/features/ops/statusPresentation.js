export const statusComponents = [
  { id: "api", title: "API", okHint: "Отвечает", badHint: "Не отвечает" },
  { id: "worker", title: "Обработчик бланков", okHint: "Отвечает", badHint: "Не отвечает" },
  { id: "postgres", title: "База", okHint: "Отвечает", badHint: "Не отвечает" },
  { id: "queue", title: "Очередь", okHint: "Отвечает", badHint: "Не отвечает" },
  { id: "files", title: "Файлы", okHint: "Отвечает", badHint: "Не отвечает" },
];

export function presentStatus(components = []) {
  const byId = new Map((components || []).map((item) => [item.id, item]));
  return statusComponents.map((meta) => {
    const raw = byId.get(meta.id);
    const known = Boolean(raw);
    const ok = Boolean(raw?.ok);
    return {
      id: meta.id,
      title: meta.title,
      ok,
      known,
      hint: !known ? "Проверяю…" : ok ? meta.okHint : meta.badHint,
    };
  });
}

export function statusHeadline(tiles) {
  if (!tiles.length || tiles.some((tile) => !tile.known)) return "Проверяю сервисы";
  if (tiles.every((tile) => tile.ok)) return "Все сервисы отвечают";
  const down = tiles.filter((tile) => !tile.ok).map((tile) => tile.title);
  if (down.length === 1) return `${down[0]} не отвечает`;
  return "Часть сервисов не отвечает";
}
