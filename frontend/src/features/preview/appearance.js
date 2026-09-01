const BORDER = "1px solid #b4b4b4";

export function cellCss(catalog, index) {
  const style = catalog?.[index];
  if (!style) return undefined;
  const css = {};
  if (style.fill) css.backgroundColor = style.fill;
  if (style.color) css.color = style.color;
  if (style.bold) css.fontWeight = 700;
  if (style.italic) css.fontStyle = "italic";
  if (style.size) css.fontSize = `${style.size}pt`;
  if (style.align === "center") css.justifyContent = "center";
  else if (style.align === "right") css.justifyContent = "flex-end";
  else if (style.align === "left") css.justifyContent = "flex-start";
  if (style.valign === "top") css.alignItems = "flex-start";
  else if (style.valign === "bottom") css.alignItems = "flex-end";
  else if (style.valign === "middle") css.alignItems = "center";
  if (style.wrap) {
    css.whiteSpace = "pre-wrap";
    css.overflow = "hidden";
    if (!css.alignItems) css.alignItems = "flex-start";
  }
  if (style.border_t) css.borderTop = BORDER;
  if (style.border_r) css.borderRight = BORDER;
  if (style.border_l) css.borderLeft = BORDER;
  if (style.border_b) css.borderBottom = BORDER;
  return Object.keys(css).length ? css : undefined;
}

export function mergeLayout(merges) {
  const covered = new Set();
  const origins = new Set();
  const list = [];
  for (const merge of merges || []) {
    const row = Number(merge.row);
    const column = Number(merge.column);
    const height = Math.max(1, Number(merge.height) || 1);
    const width = Math.max(1, Number(merge.width) || 1);
    if (row < 1 || column < 1 || (height < 2 && width < 2)) continue;
    list.push({ row, column, height, width });
    origins.add(cellKey(row, column));
    for (let currentRow = row; currentRow < row + height; currentRow += 1) {
      for (let currentCol = column; currentCol < column + width; currentCol += 1) {
        if (currentRow === row && currentCol === column) continue;
        covered.add(cellKey(currentRow, currentCol));
      }
    }
  }
  return { covered, origins, list };
}

export function cellKey(row, column) {
  return `${row}:${column}`;
}

export function visibleMerges(merges, fromRow, toRow) {
  return (merges || []).filter((merge) => {
    const end = merge.row + merge.height - 1;
    return end >= fromRow && merge.row <= toRow;
  });
}