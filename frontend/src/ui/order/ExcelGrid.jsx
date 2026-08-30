import { useCallback, useEffect, useRef, useState } from "react";

import { getPreviewWindow } from "../../api/preview.js";
import { columnLetters } from "../../features/preview/columns.js";
import {
  PREVIEW_COL_WIDTH,
  PREVIEW_GUTTER_WIDTH,
  PREVIEW_HEADER_HEIGHT,
  PREVIEW_MAX_FETCH_ROWS,
  PREVIEW_ROW_HEIGHT,
  missingRange,
  scrollTopForRow,
  visibleWindow,
} from "../../features/preview/viewport.js";

export function ExcelGrid({ jobId, fileId, sheetIndex = 0, maxRow, maxColumn, headerRow, highlightRow, focusRow }) {
  const scrollerRef = useRef(null);
  const cacheRef = useRef(new Map());
  const fetchRef = useRef(0);
  const timerRef = useRef(0);
  const [cells, setCells] = useState(() => new Map());
  const [range, setRange] = useState({ fromRow: 1, toRow: 1 });
  const letters = columnLetters(maxColumn);
  const gridWidth = PREVIEW_GUTTER_WIDTH + letters.length * PREVIEW_COL_WIDTH;

  const syncWindow = useCallback(() => {
    const node = scrollerRef.current;
    if (!node || !maxRow) return;
    const next = visibleWindow({
      scrollTop: node.scrollTop,
      viewportHeight: node.clientHeight || 640,
      maxRow,
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
        for (let index = 0; index < rows.length; index += 1) {
          cacheRef.current.set(from + index, rows[index]);
        }
        setCells(new Map(cacheRef.current));
      })
      .catch(() => {
        // Keep already painted rows; the next scroll retries the window.
      });
  }, [fileId, jobId, maxRow, sheetIndex]);

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
  }, [fileId, jobId, sheetIndex, maxRow, maxColumn, syncWindow]);

  useEffect(() => {
    if (!focusRow || !scrollerRef.current) return;
    scrollerRef.current.scrollTop = scrollTopForRow(focusRow);
    syncWindow();
  }, [focusRow, syncWindow]);

  function onScroll() {
    window.clearTimeout(timerRef.current);
    timerRef.current = window.setTimeout(syncWindow, 40);
  }

  const rows = [];
  for (let row = range.fromRow; row <= range.toRow; row += 1) {
    rows.push(row);
  }

  return (
    <div
      ref={scrollerRef}
      onScroll={onScroll}
      className="h-full overflow-auto bg-[var(--color-surface)]"
    >
      <div style={{ width: gridWidth, minWidth: "100%" }}>
        <div
          className="sticky top-0 z-20 flex border-b border-[var(--color-line)] bg-[var(--color-ground)]"
          style={{ height: PREVIEW_HEADER_HEIGHT }}
        >
          <div
            className="sticky left-0 z-30 border-r border-[var(--color-line)] bg-[var(--color-ground)]"
            style={{ width: PREVIEW_GUTTER_WIDTH, minWidth: PREVIEW_GUTTER_WIDTH }}
          />
          {letters.map((letter) => (
            <div
              key={letter}
              className="flex shrink-0 items-center justify-center border-r border-[var(--color-line-soft)] font-mono text-[11px] font-medium text-[var(--color-ink-faint)]"
              style={{ width: PREVIEW_COL_WIDTH }}
            >
              {letter}
            </div>
          ))}
        </div>
        <div className="relative" style={{ height: Math.max(maxRow, 1) * PREVIEW_ROW_HEIGHT }}>
          {rows.map((row) => (
            <GridRow
              key={row}
              row={row}
              values={cells.get(row)}
              columns={letters.length}
              headerRow={headerRow}
              highlight={row === highlightRow}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function GridRow({ row, values, columns, headerRow, highlight }) {
  const isHeader = row === headerRow;
  return (
    <div
      className={`absolute left-0 flex border-b border-[var(--color-line-soft)] ${
        highlight ? "bg-[var(--color-brand-soft)]" : isHeader ? "bg-[var(--color-line-soft)]" : "bg-[var(--color-surface)]"
      }`}
      style={{ top: (row - 1) * PREVIEW_ROW_HEIGHT, height: PREVIEW_ROW_HEIGHT, width: "100%" }}
    >
      <div
        className="sticky left-0 z-10 flex items-center justify-end border-r border-[var(--color-line)] bg-inherit px-2 font-mono text-[11px] text-[var(--color-ink-faint)]"
        style={{ width: PREVIEW_GUTTER_WIDTH, minWidth: PREVIEW_GUTTER_WIDTH }}
      >
        {row}
      </div>
      {Array.from({ length: columns }, (_, index) => {
        const value = values?.[index] ?? "";
        return (
          <div
            key={index}
            title={value}
            className={`flex shrink-0 items-center overflow-hidden border-r border-[var(--color-line-soft)] px-2 text-[12px] leading-none ${
              isHeader ? "font-semibold text-[var(--color-ink)]" : "text-[var(--color-ink-soft)]"
            }`}
            style={{ width: PREVIEW_COL_WIDTH }}
          >
            <span className="truncate">{value}</span>
          </div>
        );
      })}
    </div>
  );
}
