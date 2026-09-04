import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { getPreviewWindow } from "../../api/preview.js";
import { cellCss, cellKey, mergeLayout, visibleMerges } from "../../features/preview/appearance.js";
import { columnLetters } from "../../features/preview/columns.js";
import { isCommentOverlay, isQuantityOverlay } from "../../features/preview/previewEdits.js";
import {
  PREVIEW_GUTTER_WIDTH,
  PREVIEW_HEADER_HEIGHT,
  PREVIEW_MAX_FETCH_ROWS,
  PREVIEW_ROW_HEIGHT,
  buildRowOffsets,
  columnOffsets,
  columnSize,
  gridContentWidth,
  missingRange,
  parseCustomHeights,
  scrollTopForRow,
  sheetHeight,
  spanSize,
  visibleWindow,
} from "../../features/preview/viewport.js";

const GRID_LINE = "1px solid var(--color-line-soft)";

export function ExcelGrid({
  jobId,
  fileId,
  sheetIndex = 0,
  maxRow,
  maxColumn,
  headerRow,
  highlightRow,
  focusRow,
  columns,
  rowHeight,
  rowHeights,
  styles: catalog,
  merges,
  overlays,
  onEdit,
  refreshKey = 0,
}) {
  const scrollerRef = useRef(null);
  const cacheRef = useRef(new Map());
  const fetchRef = useRef(0);
  const timerRef = useRef(0);
  const [cells, setCells] = useState(() => new Map());
  const [range, setRange] = useState({ fromRow: 1, toRow: 1 });
  const [activeKey, setActiveKey] = useState("");
  const letters = columnLetters(maxColumn);
  const defaultHeight = Number(rowHeight) > 0 ? Number(rowHeight) : PREVIEW_ROW_HEIGHT;
  const customHeights = useMemo(() => parseCustomHeights(rowHeights), [rowHeights]);
  const offsets = useMemo(
    () => buildRowOffsets(maxRow, defaultHeight, customHeights),
    [customHeights, defaultHeight, maxRow],
  );
  const colOffsets = useMemo(() => columnOffsets(maxColumn, columns), [columns, maxColumn]);
  const mergeMap = useMemo(() => mergeLayout(merges), [merges]);
  const gridWidth = gridContentWidth(maxColumn, columns);
  const bodyHeight = sheetHeight(offsets, maxRow, defaultHeight);

  const syncWindow = useCallback(() => {
    const node = scrollerRef.current;
    if (!node || !maxRow) return;
    const next = visibleWindow({
      scrollTop: node.scrollTop,
      viewportHeight: node.clientHeight || 640,
      maxRow,
      offsets,
      rowHeight: defaultHeight,
    });
    setRange((prev) => (prev.fromRow === next.fromRow && prev.toRow === next.toRow ? prev : next));
    const missing = missingRange(cacheRef.current, next.fromRow, next.toRow);
    if (!missing) return;
    const toRow = Math.min(missing.toRow, missing.fromRow + PREVIEW_MAX_FETCH_ROWS - 1);
    const requestId = (fetchRef.current += 1);
    getPreviewWindow(jobId, fileId, { sheet: sheetIndex, fromRow: missing.fromRow, toRow })
      .then((payload) => {
        if (requestId !== fetchRef.current) return;
        const from = payload.from_row;
        const rows = payload.rows || [];
        const styleRows = payload.styles || [];
        for (let index = 0; index < rows.length; index += 1) {
          cacheRef.current.set(from + index, { values: rows[index], styles: styleRows[index] });
        }
        setCells(new Map(cacheRef.current));
      })
      .catch(() => {
        // Keep already painted rows; the next scroll retries the window.
      });
  }, [defaultHeight, fileId, jobId, maxRow, offsets, sheetIndex]);

  useEffect(() => {
    cacheRef.current = new Map();
    setCells(new Map());
    fetchRef.current += 1;
    const node = scrollerRef.current;
    if (node) node.scrollTop = 0;
    const frame = window.requestAnimationFrame(syncWindow);
    const observer = node ? new ResizeObserver(syncWindow) : null;
    if (node && observer) observer.observe(node);
    return () => {
      window.cancelAnimationFrame(frame);
      window.clearTimeout(timerRef.current);
      observer?.disconnect();
    };
  }, [fileId, jobId, sheetIndex, maxRow, maxColumn, refreshKey, syncWindow]);

  useEffect(() => {
    if (!focusRow || !scrollerRef.current) return;
    scrollerRef.current.scrollTop = scrollTopForRow(focusRow, defaultHeight, offsets);
    syncWindow();
  }, [defaultHeight, focusRow, offsets, syncWindow]);

  function onScroll() {
    window.clearTimeout(timerRef.current);
    timerRef.current = window.setTimeout(syncWindow, 40);
  }

  const rows = [];
  for (let row = range.fromRow; row <= range.toRow; row += 1) {
    rows.push(row);
  }
  const mergeBoxes = visibleMerges(mergeMap.list, range.fromRow, range.toRow);

  return (
    <div
      ref={scrollerRef}
      onScroll={onScroll}
      className="h-full overflow-auto bg-[var(--color-surface)]"
    >
      <div style={{ width: gridWidth, position: "relative" }}>
        <div
          className="sticky top-0 z-20 border-b border-[var(--color-line)] bg-[var(--color-ground)]"
          style={{ height: PREVIEW_HEADER_HEIGHT, width: gridWidth }}
        >
          <div
            className="sticky left-0 z-30 box-border border-r border-[var(--color-line)] bg-[var(--color-ground)]"
            style={{
              position: "sticky",
              left: 0,
              top: 0,
              width: PREVIEW_GUTTER_WIDTH,
              height: PREVIEW_HEADER_HEIGHT,
            }}
          />
          {letters.map((letter, index) => {
            const width = columnSize(index, columns);
            if (width <= 0) return null;
            return (
              <div
                key={letter}
                className="absolute top-0 box-border flex items-center justify-center border-r border-[var(--color-line-soft)] font-mono text-[11px] font-medium text-[var(--color-ink-faint)]"
                style={{
                  left: PREVIEW_GUTTER_WIDTH + colOffsets[index],
                  width,
                  height: PREVIEW_HEADER_HEIGHT,
                }}
              >
                {letter}
              </div>
            );
          })}
        </div>
        <div
          className="relative"
          style={{ height: bodyHeight, width: gridWidth, fontFamily: "Calibri, Arial, sans-serif" }}
        >
          {rows.map((row) => (
            <RowGutter
              key={`g-${row}`}
              row={row}
              top={offsets[row]}
              height={rowHeightOfLocal(row, defaultHeight, customHeights)}
              gridWidth={gridWidth}
            />
          ))}
          {rows.map((row) => (
            <GridRowCells
              key={row}
              row={row}
              record={cells.get(row)}
              columns={letters.length}
              widths={columns}
              colOffsets={colOffsets}
              top={offsets[row]}
              height={rowHeightOfLocal(row, defaultHeight, customHeights)}
              headerRow={headerRow}
              highlight={row === highlightRow}
              covered={mergeMap.covered}
              origins={mergeMap.origins}
              catalog={catalog}
              overlays={overlays}
              activeKey={activeKey}
              onActivate={setActiveKey}
              onEdit={onEdit}
            />
          ))}
          {mergeBoxes.map((merge) => {
            const record = cells.get(merge.row);
            const value = record?.values?.[merge.column - 1] ?? "";
            const css = cellCss(catalog, record?.styles?.[merge.column - 1] ?? 0);
            const highlighted = highlightRow >= merge.row && highlightRow < merge.row + merge.height;
            const width = spanSize(colOffsets, merge.column - 1, merge.width);
            const height = spanSize(offsets, merge.row, merge.height);
            if (width <= 0 || height <= 0) return null;
            return (
              <GridCell
                key={cellKey(merge.row, merge.column)}
                left={PREVIEW_GUTTER_WIDTH + colOffsets[merge.column - 1]}
                top={offsets[merge.row]}
                width={width}
                height={height}
                value={value}
                css={css}
                highlight={highlighted}
                header={merge.row === headerRow && !css?.backgroundColor}
                z={2}
                overlay={overlays?.get(`${merge.row}:${merge.column}`)}
                active={activeKey === overlays?.get(`${merge.row}:${merge.column}`)?.key}
                onActivate={setActiveKey}
                onEdit={onEdit}
              />
            );
          })}
        </div>
      </div>
    </div>
  );
}

function rowHeightOfLocal(row, defaultHeight, customHeights) {
  const custom = customHeights.get(row);
  return custom > 0 ? custom : defaultHeight;
}

function RowGutter({ row, top, height, gridWidth }) {
  return (
    <div
      className="pointer-events-none absolute left-0"
      style={{ top, height, width: gridWidth }}
    >
      <div
        className="pointer-events-auto sticky left-0 z-10 box-border flex items-center justify-end border-r border-b border-[var(--color-line)] bg-[var(--color-ground)] px-2 font-mono text-[11px] text-[var(--color-ink-faint)]"
        style={{ width: PREVIEW_GUTTER_WIDTH, height }}
      >
        {row}
      </div>
    </div>
  );
}

function GridRowCells({
  row,
  record,
  columns,
  widths,
  colOffsets,
  top,
  height,
  headerRow,
  highlight,
  covered,
  origins,
  catalog,
  overlays,
  activeKey,
  onActivate,
  onEdit,
}) {
  const isHeader = row === headerRow;
  const values = record?.values;
  const styles = record?.styles;
  const cells = [];
  for (let index = 0; index < columns; index += 1) {
    const key = cellKey(row, index + 1);
    if (covered.has(key) || origins.has(key)) continue;
    const width = columnSize(index, widths);
    if (width <= 0) continue;
    const css = cellCss(catalog, styles?.[index] ?? 0);
    const overlay = overlays?.get(`${row}:${index + 1}`);
    cells.push(
      <GridCell
        key={index}
        left={PREVIEW_GUTTER_WIDTH + colOffsets[index]}
        top={top}
        width={width}
        height={height}
        value={values?.[index] ?? ""}
        css={css}
        highlight={highlight}
        header={isHeader && !css?.backgroundColor}
        overlay={overlay}
        active={overlay?.key && overlay.key === activeKey}
        onActivate={onActivate}
        onEdit={onEdit}
      />,
    );
  }
  return cells;
}

function GridCell({
  left,
  top,
  width,
  height,
  value,
  css,
  highlight,
  header,
  z = 0,
  overlay,
  active,
  onActivate,
  onEdit,
}) {
  const quantity = isQuantityOverlay(overlay);
  const commentCell = isCommentOverlay(overlay);
  const editable = Boolean((quantity || commentCell) && onEdit);
  const shown = quantity || commentCell ? overlay.value : overlay?.value || value;
  const comment = quantity ? String(overlay?.comment || "").trim() : "";

  return (
    <div
      title={comment || shown}
      className={`absolute box-border flex items-center px-2 text-[12px] leading-tight ${
        header ? "font-semibold text-[var(--color-ink)]" : "text-[var(--color-ink-soft)]"
      } ${editable ? "overflow-visible" : "overflow-hidden"}`}
      style={{
        left,
        top,
        width,
        height,
        borderRight: GRID_LINE,
        borderBottom: GRID_LINE,
        ...css,
        zIndex: editable ? (active ? 8 : 3) : active ? 8 : z,
        overflow: editable ? "visible" : css?.overflow,
        backgroundColor: editable
          ? "color-mix(in srgb, var(--color-brand-soft) 85%, var(--color-surface))"
          : header
            ? "var(--color-line-soft)"
            : css?.backgroundColor || "var(--color-surface)",
        boxShadow: editable
          ? `inset 0 0 0 ${active ? 2 : 1}px var(--color-brand)`
          : highlight || active
            ? "inset 0 0 0 2px var(--color-brand)"
            : undefined,
      }}
    >
      {comment ? <CommentMark /> : null}
      {quantity ? (
        <input
          value={overlay.value}
          aria-label="Заказано по факту"
          onFocus={() => onActivate?.(overlay.key)}
          onBlur={(event) => {
            if (event.currentTarget.parentElement?.contains(event.relatedTarget)) return;
            onActivate?.("");
          }}
          onChange={(event) => onEdit(overlay.key, { value: event.target.value })}
          className="h-full w-full cursor-text bg-transparent font-mono text-[12px] text-[var(--color-ink)] outline-none"
        />
      ) : commentCell ? (
        <input
          value={overlay.value}
          aria-label="Комментарий"
          onChange={(event) => onEdit(overlay.key, { comment: event.target.value })}
          className="h-full w-full cursor-text bg-transparent text-[12px] text-[var(--color-ink)] outline-none"
        />
      ) : (
        <span className={css?.whiteSpace ? "w-full" : "truncate"}>{shown}</span>
      )}
      {quantity && active ? (
        <textarea
          value={overlay.comment || ""}
          placeholder="Почему изменили количество"
          onFocus={() => onActivate?.(overlay.key)}
          onBlur={(event) => {
            if (event.currentTarget.parentElement?.contains(event.relatedTarget)) return;
            onActivate?.("");
          }}
          onChange={(event) => onEdit(overlay.key, { comment: event.target.value })}
          className="absolute left-0 top-full z-30 mt-1 w-64 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] p-2 text-[13px] text-[var(--color-ink)] shadow-lg outline-none"
          rows={2}
        />
      ) : null}
    </div>
  );
}

function CommentMark() {
  return (
    <span
      aria-hidden
      className="pointer-events-none absolute right-0 top-0 h-0 w-0 border-l-8 border-t-8 border-l-transparent border-t-[#ea580c]"
    />
  );
}
