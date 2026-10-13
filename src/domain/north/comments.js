import { demandPiecesFromSupplierUnits } from "./../order/quantities.js";
import { NORTH_CITIES, NORTH_TRANSFER_DISPLAY_ORDER } from "./cities.js";
import { normalizeName } from "./../matching/identity.js";

export function formatNorthQuantity(value) {
  const number = Number(value || 0);
  if (Number.isInteger(number)) return String(number);
  return String(Number(number.toFixed(2)));
}

export function formatNorthCommentQuantity(value) {
  const number = Math.round(Number(value || 0));
  return Number.isFinite(number) && number > 0 ? String(number) : "";
}

export function supplierPartsForActual(row, actualValue) {
  const actual = demandPiecesFromSupplierUnits(actualValue, row.supplierUnitSize);
  const northParts = (row.supplierParts || []).filter((part) => part.key !== "tyumen");
  const northNeed = northParts.reduce((sum, part) => sum + Number(part.quantity || 0), 0);
  const tyumenQuantity = Number(Math.max(0, actual - northNeed).toFixed(2));
  return [
    ...(tyumenQuantity > 0 ? [{ key: "tyumen", label: "Тюмень", quantity: tyumenQuantity }] : []),
    ...northParts,
  ];
}

export function northTransferParts(row) {
  const quantities = new Map((row.cities || []).map((city) => [city.key, Number(city.quantity || 0)]));
  return NORTH_TRANSFER_DISPLAY_ORDER
    .map((cityKey) => {
      const city = NORTH_CITIES.find((item) => item.key === cityKey);
      return city ? { key: city.key, label: city.label, quantity: quantities.get(city.key) || 0 } : null;
    })
    .filter(Boolean);
}

export function northSupplierOrderText(row, actual) {
  const actualRounded = Math.round(Number(actual || 0));
  const neededRounded = Math.round(Number(row.supplierNeed || 0));
  const extraRounded = Math.max(0, actualRounded - neededRounded);
  const unitNote = Number(row.supplierUnitSize || 1) > 1 ? ` коробок по ${Number(row.supplierUnitSize)}` : "";
  if (extraRounded > 0) {
    return `${neededRounded} + ${extraRounded} (${row.budgetComment ? 'сверх потребности' : 'до минимального'}) = ${actualRounded}${unitNote}`;
  }
  return `${formatNorthQuantity(actual)}${unitNote}`;
}

export function northPlanCommentForOrderTable(row, actualValue) {
  const lines = [];
  const actual = Number(actualValue || 0);
  const supplierParts = supplierPartsForActual(row, actualValue);
  const tyumenSupplier = supplierParts.find((part) => part.key === "tyumen");
  const transferParts = northTransferParts(row);

  if (actual > 0) lines.push(`Заказать у поставщика: ${northSupplierOrderText(row, actual)}`);
  for (const part of transferParts) {
    if (Number(part.quantity || 0) > 0) lines.push(`Отправить в ${part.label}: ${formatNorthQuantity(part.quantity)}`);
  }
  if (Number(tyumenSupplier?.quantity || 0) > 0) {
    lines.push(`Оставить в Тюмени: ${formatNorthCommentQuantity(tyumenSupplier.quantity)}`);
  }
  if (!lines.length && Number(row.northNeed || 0) > 0) lines.push("Закрывается остатком Тюмени");
  if (row.budgetComment) lines.push(row.budgetComment);
  return lines.join("\n");
}

export function novacutanNorthOrderTable(summary, planRows, actualByKey) {
  if (summary.kind !== "novacutan" && !planRows.some(r => r.budgetComment)) return null;
  const rows = [];

  for (const row of planRows) {
    const orderedQuantity = Number(actualByKey.get(row.key) || 0);
    if (orderedQuantity == null) continue;
    if (orderedQuantity <= 0) continue;

    rows.push({
      name: row.name,
      quantity: orderedQuantity,
      comment: northPlanCommentForOrderTable(row, orderedQuantity),
    });
  }

  return {
    fileName: `${summary.kind === 'novacutan' ? 'NOVACUTAN' : 'Север'} заполненная таблица.xlsx`,
    rows,
  };
}

export function northNameKey(name, unit = "") {
  const normalized = normalizeName(`${name} ${unit || ""}`);
  return normalized ? `name:${normalized}` : "";
}
