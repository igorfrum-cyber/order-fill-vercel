export function normalizeOrderValue(value) {
  const text = String(value ?? "").trim().replace(",", ".");
  if (text === "") return null;
  const number = Number(text);
  if (!Number.isFinite(number) || number < 0) {
    throw new Error("Количество должно быть неотрицательным числом.");
  }
  return number;
}

export function editRequiresComment({ value, baseline, initial, comment, autoComment }) {
  const normalizedComment = String(comment || "").trim().toLowerCase();
  const normalizedAutoComment = String(autoComment || "").trim().toLowerCase();
  const stillAutoComment = Boolean(normalizedAutoComment) && normalizedComment === normalizedAutoComment;
  const autoCommentAllowed = stillAutoComment && value === initial;
  return value !== baseline && (!normalizedComment || (stillAutoComment && !autoCommentAllowed));
}
