import { PREVIEW_HEADER_HEIGHT, PREVIEW_ROW_HEIGHT } from "./viewport.js";

export function isHeaderPinned({ freeze, headerRow, scrollTop, offsets } = {}) {
  if (!freeze) return false;
  const row = Math.floor(Number(headerRow) || 0);
  if (row < 1) return false;
  const headerTop = Number(offsets?.[row]);
  const top = Number.isFinite(headerTop) ? headerTop : 0;
  return (Number(scrollTop) || 0) >= top;
}

export function freezeChromeHeight({
  pinned,
  letterHeight = PREVIEW_HEADER_HEIGHT,
  headerRowHeight = PREVIEW_ROW_HEIGHT,
} = {}) {
  const letters = Number(letterHeight) > 0 ? Number(letterHeight) : 0;
  if (!pinned) return letters;
  const header = Number(headerRowHeight) > 0 ? Number(headerRowHeight) : 0;
  return letters + header;
}
