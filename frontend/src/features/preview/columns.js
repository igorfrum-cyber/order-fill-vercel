export function columnName(column) {
  let name = "";
  let current = Number(column) || 0;
  while (current > 0) {
    const remainder = (current - 1) % 26;
    name = String.fromCharCode(65 + remainder) + name;
    current = Math.floor((current - 1) / 26);
  }
  return name;
}

export const PREVIEW_MAX_COLUMNS = 128;

export function visibleColumnCount(maxColumn) {
  const count = Math.max(0, Math.floor(Number(maxColumn) || 0));
  return Math.min(count, PREVIEW_MAX_COLUMNS);
}

export function columnLetters(maxColumn) {
  const count = visibleColumnCount(maxColumn);
  const letters = [];
  for (let column = 1; column <= count; column += 1) {
    letters.push(columnName(column));
  }
  return letters;
}
