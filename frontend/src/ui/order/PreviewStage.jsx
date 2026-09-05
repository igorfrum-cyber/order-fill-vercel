import { useEffect, useMemo, useRef, useState } from "react";

import { findPreviewArticle, getPreviewMeta, getPreviewWindow } from "../../api/preview.js";
import { userFacingError } from "../../features/help/errors.js";
import { columnName } from "../../features/preview/columns.js";
import { previewFileTitle } from "../../features/preview/fileTitle.js";
import { formulaOverlays } from "../../features/preview/formulas.js";
import {
  previewBodyState,
  previewEmptyHint,
  previewLoadingHint,
  previewLoadingTitle,
} from "../../features/preview/previewStatus.js";
import {
  defaultPreviewFileId,
  findEditColumns,
  isSourcePreviewFile,
  mergePreviewOverlays,
  needsHeaderScan,
  orderSheetIndex,
  previewOverlays,
} from "../../features/preview/previewEdits.js";
import { ErrorBoundary } from "../ErrorBoundary.jsx";
import { IconDownload, IconPin, IconSearch, IconX } from "../icons.jsx";
import { GhostButton, PrimaryButton, ProgressBar } from "../widgets.jsx";
import { ExcelGrid } from "./ExcelGrid.jsx";

export function PreviewStage({
  files = [],
  jobId,
  status,
  busy,
  rows = [],
  edits,
  onEdit,
  refreshKey = 0,
  onDownload,
  onBack,
  onReady,
}) {
  const defaultFileId = defaultPreviewFileId(files);
  const [fileId, setFileId] = useState(defaultFileId);
  const [meta, setMeta] = useState(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [highlightRow, setHighlightRow] = useState(0);
  const [focusRow, setFocusRow] = useState(0);
  const [sheetIndex, setSheetIndex] = useState(0);
  const [findStatus, setFindStatus] = useState("");
  const [headerCells, setHeaderCells] = useState([]);
  const [gridReady, setGridReady] = useState(false);
  const [overlays, setOverlays] = useState(() => new Map());
  const [freezeHeader, setFreezeHeader] = useState(true);
  const workAreaRef = useRef(null);
  const autoScrolledRef = useRef(false);

  const file = files.find((item) => item.id === fileId) || files[0];
  const sheets = meta?.sheets || [];
  const sheet = sheets[sheetIndex] || sheets[0];

  useEffect(() => {
    if (!files.length) return;
    if (files.some((item) => item.id === fileId)) return;
    setFileId(defaultPreviewFileId(files));
  }, [fileId, files]);

  useEffect(() => {
    if (!jobId || !file?.id) return;
    let cancelled = false;
    setError("");
    setMeta(null);
    setSheetIndex(0);
    setHighlightRow(0);
    setFocusRow(0);
    setHeaderCells([]);
    setGridReady(false);
    getPreviewMeta(jobId, file.id)
      .then((payload) => {
        if (cancelled) return;
        setMeta(payload);
        setSheetIndex(orderSheetIndex(payload.sheets || []));
      })
      .catch((err) => {
        if (!cancelled) setError(userFacingError(err, "Не удалось загрузить превью."));
      });
    return () => {
      cancelled = true;
    };
  }, [file?.id, jobId, refreshKey]);

  useEffect(() => {
    autoScrolledRef.current = false;
  }, [file?.id, jobId, refreshKey, sheetIndex]);

  const sourceFile = isSourcePreviewFile(file);
  const editColumns = useMemo(() => {
    if (sheet?.quantity_column) {
      return { quantity: sheet.quantity_column, comment: sheet.comment_column || 0 };
    }
    return findEditColumns(headerCells);
  }, [headerCells, sheet?.comment_column, sheet?.quantity_column]);

  useEffect(() => {
    if (!needsHeaderScan(sheet, { sourceFile, jobId, fileId: file?.id })) return;
    const headerRow = Number(sheet.header_row);
    let cancelled = false;
    getPreviewWindow(jobId, file.id, {
      sheet: sheet.index ?? sheetIndex,
      fromRow: headerRow,
      toRow: headerRow,
    })
      .then((payload) => {
        if (!cancelled) setHeaderCells(payload.rows?.[0] || []);
      })
      .catch(() => {
        if (!cancelled) setHeaderCells([]);
      });
    return () => {
      cancelled = true;
    };
  }, [file?.id, jobId, sheet, sheetIndex, sourceFile]);

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      try {
        const quantity = previewOverlays(rows, edits instanceof Map ? edits : new Map(), {
          files,
          fileId: file?.id,
          quantityColumn: editColumns.quantity,
          commentColumn: editColumns.comment,
        });
        setOverlays(
          mergePreviewOverlays(
            formulaOverlays(sheet?.formulas, { overlays: quantity, values: sheet?.formula_values || {} }),
            quantity,
          ),
        );
      } catch {
        setOverlays(new Map());
      }
    });
    return () => window.cancelAnimationFrame(frame);
  }, [editColumns.comment, editColumns.quantity, edits, file?.id, files, rows, sheet?.formula_values, sheet?.formulas]);

  const bodyState = previewBodyState({
    error,
    fileId: file?.id,
    meta,
    sheet,
    gridReady,
  });

  useEffect(() => {
    if (bodyState !== "ready" || autoScrolledRef.current) return undefined;
    autoScrolledRef.current = true;
    const frame = window.requestAnimationFrame(() => {
      workAreaRef.current?.scrollIntoView({
        behavior: "smooth",
        block: "start",
      });
    });
    return () => window.cancelAnimationFrame(frame);
  }, [bodyState]);

  const stats = sheet ? `${sheet.max_row} строк · до ${columnName(sheet.max_column)}` : "";
  const canFreezeHeader = Number(sheet?.header_row) > 0;

  async function jumpToArticle(event) {
    event.preventDefault();
    const needle = query.trim();
    if (!needle || !file?.id) return;
    setFindStatus("Ищу...");
    try {
      const hit = await findPreviewArticle(jobId, file.id, { sheet: sheetIndex, query: needle });
      if (!hit.found) {
        setFindStatus("Артикул не найден");
        return;
      }
      setHighlightRow(hit.row);
      setFocusRow(hit.row);
      setFindStatus(`строка ${hit.row}`);
    } catch (err) {
      setFindStatus(userFacingError(err, "Не удалось найти артикул"));
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-wrap items-center gap-3 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-3">
        <div data-tour="preview-files" className="flex flex-wrap gap-1">
          {files.map((item) => {
            const active = item.id === file?.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => setFileId(item.id)}
                className={`rounded-lg px-3 py-2 text-[14px] font-medium transition ${
                  active ? "bg-[var(--color-brand)] text-white" : "text-[var(--color-ink-soft)] hover:bg-[var(--color-line-soft)]"
                }`}
              >
                {previewFileTitle(item)}
              </button>
            );
          })}
        </div>
        <span className="text-[13px] text-[var(--color-ink-faint)]">
          Править «Заказано по факту» и комментарий — количество в бланке подтянется само
        </span>
        {sheets.length > 1 && (
          <div className="flex gap-1">
            {sheets.map((item) => (
              <button
                key={item.index}
                type="button"
                onClick={() => {
                  setGridReady(false);
                  setSheetIndex(item.index);
                }}
                className={`rounded-md px-2 py-1 font-mono text-[12px] ${
                  item.index === sheetIndex ? "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]" : "text-[var(--color-ink-faint)]"
                }`}
              >
                {item.name}
              </button>
            ))}
          </div>
        )}
        <button
          type="button"
          aria-pressed={freezeHeader && canFreezeHeader}
          disabled={!canFreezeHeader}
          onClick={() => setFreezeHeader((on) => !on)}
          className={`flex items-center gap-1.5 rounded-lg px-3 py-2 text-[14px] font-medium transition ${
            freezeHeader && canFreezeHeader
              ? "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]"
              : "text-[var(--color-ink-soft)] hover:bg-[var(--color-line-soft)]"
          } disabled:cursor-not-allowed disabled:opacity-40`}
        >
          <IconPin className="h-4 w-4" />
          {freezeHeader ? "Шапка закреплена" : "Закрепить шапку"}
        </button>
        <form onSubmit={jumpToArticle} className="relative ml-auto min-w-64">
          <IconSearch className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--color-ink-faint)]" />
          <input
            value={query}
            onChange={(event) => {
              setQuery(event.target.value);
              setFindStatus("");
            }}
            placeholder="Найти артикул"
            className="w-full rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] py-2 pl-8 pr-8 text-[14px] outline-none transition focus:border-[var(--color-brand)] focus:ring-4 focus:ring-[var(--color-brand-soft)]"
          />
          {query && (
            <button type="button" onClick={() => setQuery("")} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--color-ink-faint)] hover:text-[var(--color-ink)]">
              <IconX className="h-4 w-4" />
            </button>
          )}
        </form>
        <span className="font-mono text-[12px] text-[var(--color-ink-faint)]">{findStatus || stats}</span>
      </div>

      <div ref={workAreaRef} className="relative min-h-0 flex-1">
        {file?.id && sheet ? (
          <div className="absolute inset-0">
            <ErrorBoundary
              key={`${file.id}:${sheet.index ?? sheetIndex}:${refreshKey}`}
              fallback={(err) => (
                <div className="grid h-full place-items-center bg-[var(--color-ground)] px-6">
                  <div className="max-w-md text-center">
                    <p className="text-[16px] leading-relaxed text-[var(--color-danger)]">
                      {userFacingError(err, "Не удалось нарисовать сетку файла.")}
                    </p>
                    {err?.message ? (
                      <p className="mt-3 break-all font-mono text-[12px] text-[var(--color-ink-faint)]">{String(err.message)}</p>
                    ) : null}
                  </div>
                </div>
              )}
            >
              <ExcelGrid
                jobId={jobId}
                fileId={file.id}
                sheetIndex={sheet.index ?? sheetIndex}
                maxRow={sheet.max_row}
                maxColumn={sheet.max_column}
                headerRow={sheet.header_row}
                freezeHeader={canFreezeHeader && freezeHeader}
                highlightRow={highlightRow}
                focusRow={focusRow}
                columns={sheet.columns}
                rowHeight={sheet.row_height}
                rowHeights={sheet.row_heights}
                styles={sheet.styles}
                merges={sheet.merges}
                overlays={overlays instanceof Map ? overlays : new Map()}
                onEdit={onEdit}
                onReady={() => {
                  setGridReady(true);
                  onReady?.();
                }}
                onError={(err) => setError(userFacingError(err, "Не удалось загрузить сетку."))}
                refreshKey={refreshKey}
              />
            </ErrorBoundary>
          </div>
        ) : null}
        {bodyState === "error" ? (
          <div className="absolute inset-0 z-10 grid place-items-center bg-[var(--color-ground)] px-6">
            <p className="max-w-md text-center text-[16px] leading-relaxed text-[var(--color-danger)]">{error}</p>
          </div>
        ) : null}
        {bodyState === "empty" ? (
          <div className="absolute inset-0 z-10 grid place-items-center bg-[var(--color-ground)] px-6">
            <p className="max-w-md text-center text-[16px] leading-relaxed text-[var(--color-ink-soft)]">{previewEmptyHint}</p>
          </div>
        ) : null}
        {bodyState === "loading" ? (
          <div className="absolute inset-0 z-10 grid place-items-center bg-[var(--color-ground)] px-6">
            <div className="w-full max-w-md text-center">
              <h2 className="text-[22px] font-semibold tracking-tight">{previewLoadingTitle}</h2>
              <p className="mt-2 text-[15px] leading-relaxed text-[var(--color-ink-soft)]">{previewLoadingHint}</p>
              <ProgressBar indeterminate label="Загружаю файл" />
            </div>
          </div>
        ) : null}
      </div>

      <footer className="flex flex-wrap items-center gap-3 border-t border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-3">
        <GhostButton onClick={onBack} disabled={busy}>
          Назад к правкам
        </GhostButton>
        <div className="ml-auto flex items-center gap-3">
          <span className="font-mono text-[13px] text-[var(--color-ink-soft)]">{status}</span>
          <PrimaryButton dataTour="preview-download" onClick={onDownload} disabled={busy}>
            {busy ? "Готовлю файлы..." : "Скачать файлы"}
            <IconDownload className="h-4 w-4" />
          </PrimaryButton>
        </div>
      </footer>
    </div>
  );
}
