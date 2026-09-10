export const PREVIEW_ROW_HEIGHT = 28;
export const PREVIEW_COL_WIDTH = 92;
export const PREVIEW_GUTTER_WIDTH = 52;
export const PREVIEW_HEADER_HEIGHT = 28;
export const PREVIEW_BUFFER_ROWS = 12;
export const PREVIEW_MAX_FETCH_ROWS = 120;

export function parseCustomHeights(raw) {
  const heights = new Map();
  if (!raw || typeof raw !== "object") return heights;
  for (const [key, value] of Object.entries(raw)) {
    const row = Number(key);
    const height = Number(value);
    if (row > 0 && height > 0) heights.set(row, height);
  }
  return heights;
}

export function rowHeightOf(row, defaultHeight = PREVIEW_ROW_HEIGHT, customHeights) {
  const custom = customHeights instanceof Map ? customHeights.get(row) : customHeights?.[row];
  const height = Number(custom);
  return height > 0 ? height : defaultHeight;
}

export function buildRowOffsets(maxRow, defaultHeight = PREVIEW_ROW_HEIGHT, customHeights) {
  const rows = Math.max(0, Math.floor(Number(maxRow) || 0));
  const offsets = new Float64Array(rows + 2);
  let y = 0;
  for (let row = 1; row <= rows; row += 1) {
    offsets[row] = y;
    y += rowHeightOf(row, defaultHeight, customHeights);
  }
  offsets[rows + 1] = y;
  return offsets;
}

export function sheetHeight(offsets, maxRow, rowHeight = PREVIEW_ROW_HEIGHT) {
  if (offsets && offsets.length > 1) {
    return offsets[offsets.length - 1];
  }
  return Math.max(Number(maxRow) || 0, 1) * rowHeight;
}

export function columnSize(index, columns, fallback = PREVIEW_COL_WIDTH) {
  if (!columns || columns.length === 0) return fallback;
  const width = Number(columns[index]);
  if (!Number.isFinite(width) || width < 0) return fallback;
  return width;
}

export function columnOffsets(maxColumn, columns, fallback = PREVIEW_COL_WIDTH) {
  const count = Math.max(0, Math.floor(Number(maxColumn) || 0));
  const offsets = new Float64Array(count + 1);
  let x = 0;
  for (let index = 0; index < count; index += 1) {
    offsets[index] = x;
    x += columnSize(index, columns, fallback);
  }
  offsets[count] = x;
  return offsets;
}

export function gridContentWidth(maxColumn, columns, gutter = PREVIEW_GUTTER_WIDTH, fallback = PREVIEW_COL_WIDTH) {
  const count = Math.max(0, Math.floor(Number(maxColumn) || 0));
  return gutter + columnOffsets(count, columns, fallback)[count];
}

export function spanSize(offsets, start, count) {
  const from = Math.max(0, start);
  const to = Math.min(offsets.length - 1, from + count);
  const size = offsets[to] - offsets[from];
  return Number.isFinite(size) && size > 0 ? size : 0;
}

export function rowAtOffset(offsets, y, maxRow) {
  const rows = Math.max(1, Number(maxRow) || 1);
  if (!offsets || y <= 0) return 1;
  if (y >= offsets[rows]) return rows;
  let lo = 1;
  let hi = rows;
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1;
    if (offsets[mid] <= y) lo = mid;
    else hi = mid - 1;
  }
  return lo;
}

export function visibleWindow({
  scrollTop,
  viewportHeight,
  rowHeight = PREVIEW_ROW_HEIGHT,
  headerHeight = PREVIEW_HEADER_HEIGHT,
  maxRow,
  buffer = PREVIEW_BUFFER_ROWS,
  offsets,
}) {
  const rows = Math.max(0, Number(maxRow) || 0);
  if (rows === 0) return { fromRow: 1, toRow: 0 };
  const top = Number(scrollTop) || 0;
  const height = Number(viewportHeight) || 0;
  if (offsets && offsets.length > 2) {
    const firstVisible = rowAtOffset(offsets, top - headerHeight, rows);
    const lastVisible = rowAtOffset(offsets, top - headerHeight + height, rows);
    const fromRow = Math.max(1, firstVisible - buffer);
    const toRow = Math.min(rows, Math.max(fromRow, lastVisible + buffer));
    return { fromRow, toRow };
  }
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

export function scrollTopForRow(row, rowHeight = PREVIEW_ROW_HEIGHT, offsets) {
  const index = Math.max(1, Number(row) || 1);
  if (offsets) return offsets[index] || 0;
  return Math.max(0, (index - 1) * rowHeight);
}

export function scrollLeftToRevealColumn({
  column,
  colOffsets,
  gutter = PREVIEW_GUTTER_WIDTH,
  viewportWidth,
  trailingColumns = 1,
} = {}) {
  const colIndex = Math.max(1, Math.floor(Number(column) || 1)) - 1;
  const last = Math.max(0, (colOffsets?.length || 1) - 1);
  const after = Math.min(last, colIndex + 1 + Math.max(0, Number(trailingColumns) || 0));
  const rightEdge = gutter + (colOffsets?.[after] || 0);
  return Math.max(0, rightEdge - (Number(viewportWidth) || 0));
}
