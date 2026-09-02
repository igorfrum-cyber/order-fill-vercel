const MONTH_NAMES = [
  "Январь",
  "Февраль",
  "Март",
  "Апрель",
  "Май",
  "Июнь",
  "Июль",
  "Август",
  "Сентябрь",
  "Октябрь",
  "Ноябрь",
  "Декабрь",
];

const YEAR_MONTH = /^(\d{4})-(\d{2})$/;

export function parseYearMonth(value) {
  const match = YEAR_MONTH.exec(String(value || "").trim());
  if (!match) return null;
  const year = Number(match[1]);
  const month = Number(match[2]);
  if (month < 1 || month > 12) return null;
  return { year, month };
}

export function formatYearMonth({ year, month }) {
  return `${year}-${String(month).padStart(2, "0")}`;
}

export function currentYearMonth(now = new Date()) {
  return { year: now.getFullYear(), month: now.getMonth() + 1 };
}

export function addMonths({ year, month }, delta) {
  const index = year * 12 + (month - 1) + delta;
  return { year: Math.floor(index / 12), month: (index % 12) + 1 };
}

export function compareYearMonth(left, right) {
  return left.year - right.year || left.month - right.month;
}

export function defaultOrderMonth(now = new Date()) {
  return formatYearMonth(addMonths(currentYearMonth(now), 1));
}

export function formatOrderMonthLabel(value) {
  const parsed = parseYearMonth(value);
  if (!parsed) return String(value || "");
  return `${MONTH_NAMES[parsed.month - 1]} ${parsed.year}`;
}

export function isSelectableOrderMonth(value, now = new Date()) {
  const parsed = parseYearMonth(value);
  if (!parsed) return false;
  return compareYearMonth(parsed, currentYearMonth(now)) >= 0;
}

export function selectableOrderMonths(now = new Date(), horizon = 18) {
  const start = currentYearMonth(now);
  const options = [];
  for (let i = 0; i < horizon; i += 1) {
    const value = formatYearMonth(addMonths(start, i));
    options.push({ value, label: formatOrderMonthLabel(value) });
  }
  return options;
}

export function sanitizeOrderMonth(value, now = new Date()) {
  if (isSelectableOrderMonth(value, now)) return parseYearMonth(value) ? formatYearMonth(parseYearMonth(value)) : value;
  return defaultOrderMonth(now);
}
