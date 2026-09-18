// Client-side input validation for the budget dialog. This is UX guarding at the
// form boundary, not business logic — calculation-service remains the source of
// truth and re-validates the target and discount.

export function discountValue(value) {
  const text = String(value).trim().replace(/%$/, "").trim();
  if (!/^\d{1,2}(?:[.,]\d{1,2})?$/.test(text)) throw new Error("Введите скидку от 0 до 99,99%.");
  const n = Number(text.replace(",", "."));
  if (!Number.isFinite(n) || n < 0 || n >= 100) throw new Error("Скидка должна быть от 0 до 99,99%.");
  return n;
}

export function budgetTargetValue(value) {
  const text = String(value).trim().replace(/\s/g, "");
  if (!/^\d{1,12}(?:[.,]\d{1,2})?$/.test(text)) throw new Error("Введите сумму от 0 до 999 999 999 999,99 ₽.");
  return Number(text.replace(",", "."));
}
