import { asText } from "./../../shared/text.js";

export const MONTHS_RU = ["", "январь", "февраль", "март", "апрель", "май", "июнь", "июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь"];

export function addMonths(date, months) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + months, 1));
}

export function lastDayOfMonth(date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + 1, 0));
}

export function parseOrderMonth(value) {
  const match = /^(\d{4})-(\d{2})$/.exec(value || "");
  if (!match) throw new Error("Выберите месяц заказа.");
  return new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, 1));
}

export function formatDate(date) {
  return `${String(date.getUTCDate()).padStart(2, "0")}.${String(date.getUTCMonth() + 1).padStart(2, "0")}.${date.getUTCFullYear()}`;
}

export function sameDate(left, right) {
  return left && right && left.getTime() === right.getTime();
}

export function expectedPeriods(orderMonth) {
  const orderDate = parseOrderMonth(orderMonth);
  const mainStart = addMonths(orderDate, -13);
  const mainEnd = lastDayOfMonth(addMonths(orderDate, -2));
  const previousStart = mainStart;
  const previousEnd = lastDayOfMonth(addMonths(mainStart, 2));
  return {
    main: { start: mainStart, end: mainEnd },
    previous: { start: previousStart, end: previousEnd },
    label: `${MONTHS_RU[orderDate.getUTCMonth() + 1]} ${orderDate.getUTCFullYear()}`,
  };
}

export function formatPeriod(period) {
  return `${formatDate(period.start)} - ${formatDate(period.end)}`;
}

export function parsePeriodRange(text) {
  const match = /(\d{1,2})\.(\d{1,2})\.(\d{4}).*?(\d{1,2})\.(\d{1,2})\.(\d{4})/.exec(asText(text));
  if (!match) return null;
  const [, d1, m1, y1, d2, m2, y2] = match.map(Number);
  return { start: new Date(Date.UTC(y1, m1 - 1, d1)), end: new Date(Date.UTC(y2, m2 - 1, d2)) };
}

export function rangesEqual(left, right) {
  return sameDate(left?.start, right?.start) && sameDate(left?.end, right?.end);
}
