export const PREVIEW_ROW_HEIGHT = 28;
export const PREVIEW_COL_WIDTH = 92;
export const PREVIEW_GUTTER_WIDTH = 52;
export const PREVIEW_HEADER_HEIGHT = 28;
export const PREVIEW_BUFFER_ROWS = 24;
export const PREVIEW_MAX_FETCH_ROWS = 120;

export function visibleWindow({
  scrollTop,
  viewportHeight,
  rowHeight = PREVIEW_ROW_HEIGHT,
  headerHeight = PREVIEW_HEADER_HEIGHT,
  maxRow,
  buffer = PREVIEW_BUFFER_ROWS,
}) {
  const rows = Math.max(0, Number(maxRow) || 0);
  if (rows === 0) return { fromRow: 1, toRow: 0 };
  const top = Number(scrollTop) || 0;
  const height = Number(viewportHeight) || 0;
  const firstVisible = Math.floor((top - headerHeight) / rowHeight) + 1;
  const lastVisible = Math.ceil((top + height - headerHeight) / rowHeight);
  const fromRow = Math.max(1, firstVisible - buffer);
  const toRow = Math.min(rows, Math.max(fromRow, lastVisible + buffer));
  return { fromRow, toRow };
}

export function missingRange(cache, fromRow, toRow) {
  let start = 0;
  let end = 0;
  for (let row = fromRow; row <= toRow; row += 1) {
    if (cache.has(row)) continue;
    if (!start) start = row;
    end = row;
  }
  if (!start) return null;
  return { fromRow: start, toRow: end };
}

export function scrollTopForRow(row, rowHeight = PREVIEW_ROW_HEIGHT) {
  return Math.max(0, (Math.max(1, Number(row) || 1) - 1) * rowHeight);
}
