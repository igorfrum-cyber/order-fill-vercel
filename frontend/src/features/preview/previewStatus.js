export const previewLoadingTitle = "Собираю сетку файла";
export const previewLoadingHint = "Подождите: большой Excel открывается не сразу, ячейки появятся здесь.";
export const previewEmptyHint = "В этом файле нет листов, которые можно показать.";

export function previewBodyState({ error, fileId, meta, sheet, gridReady } = {}) {
  if (error) return "error";
  if (!fileId) return "loading";
  if (meta && !sheet) return "empty";
  if (!meta || !sheet || !gridReady) return "loading";
  return "ready";
}
