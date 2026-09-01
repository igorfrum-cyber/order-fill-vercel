import { useEffect, useMemo, useState } from "react";

import { findPreviewArticle, getPreviewMeta } from "../../api/preview.js";
import { columnName } from "../../features/preview/columns.js";
import { previewFileTitle } from "../../features/preview/fileTitle.js";
import { IconDownload, IconSearch, IconX } from "../icons.jsx";
import { GhostButton, PrimaryButton } from "../widgets.jsx";
import { ExcelGrid } from "./ExcelGrid.jsx";

export function PreviewStage({ files = [], jobId, status, busy, onDownload, onBack }) {
  const defaultFileId = files.at(-1)?.id || files[0]?.id || "";
  const [fileId, setFileId] = useState(defaultFileId);
  const [meta, setMeta] = useState(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [highlightRow, setHighlightRow] = useState(0);
  const [focusRow, setFocusRow] = useState(0);
  const [sheetIndex, setSheetIndex] = useState(0);
  const [findStatus, setFindStatus] = useState("");

  const file = files.find((item) => item.id === fileId) || files[0];
  const sheets = meta?.sheets || [];
  const sheet = sheets[sheetIndex] || sheets[0];

  useEffect(() => {
    if (!jobId || !file?.id) return;
    let cancelled = false;
    setError("");
    setMeta(null);
    setSheetIndex(0);
    setHighlightRow(0);
    setFocusRow(0);
    getPreviewMeta(jobId, file.id)
      .then((payload) => {
        if (!cancelled) setMeta(payload);
      })
      .catch((err) => {
        if (!cancelled) setError(err.message || "Не удалось загрузить превью.");
      });
    return () => {
      cancelled = true;
    };
  }, [file?.id, jobId]);

  const stats = useMemo(() => {
    if (!sheet) return "";
    return `${sheet.max_row} строк · до ${columnName(sheet.max_column)}`;
  }, [sheet]);

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
      setFindStatus(err.message || "Не удалось найти артикул");
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-wrap items-center gap-3 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-3">
        <div className="flex flex-wrap gap-1">
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
        {sheets.length > 1 && (
          <div className="flex gap-1">
            {sheets.map((item) => (
              <button
                key={item.index}
                type="button"
                onClick={() => setSheetIndex(item.index)}
                className={`rounded-md px-2 py-1 font-mono text-[12px] ${
                  item.index === sheetIndex ? "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]" : "text-[var(--color-ink-faint)]"
                }`}
              >
                {item.name}
              </button>
            ))}
          </div>
        )}
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

      <div className="min-h-0 flex-1">
        {error && <div className="px-6 py-10 text-center text-[15px] text-[var(--color-danger)]">{error}</div>}
        {!error && sheet && (
          <ExcelGrid
            jobId={jobId}
            fileId={file.id}
            sheetIndex={sheet.index ?? sheetIndex}
            maxRow={sheet.max_row}
            maxColumn={sheet.max_column}
            headerRow={sheet.header_row}
            highlightRow={highlightRow}
            focusRow={focusRow}
            columns={sheet.columns}
            rowHeight={sheet.row_height}
            rowHeights={sheet.row_heights}
            styles={sheet.styles}
            merges={sheet.merges}
          />
        )}
        {!error && !sheet && !meta && <div className="px-6 py-10 text-center text-[15px] text-[var(--color-ink-faint)]">Загружаю сетку...</div>}
      </div>

      <footer className="flex flex-wrap items-center gap-3 border-t border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-3">
        <GhostButton onClick={onBack} disabled={busy}>
          Назад к правкам
        </GhostButton>
        <div className="ml-auto flex items-center gap-3">
          <span className="font-mono text-[13px] text-[var(--color-ink-soft)]">{status}</span>
          <PrimaryButton onClick={onDownload} disabled={busy}>
            {busy ? "Готовлю архив..." : "Скачать zip"}
            <IconDownload className="h-4 w-4" />
          </PrimaryButton>
        </div>
      </footer>
    </div>
  );
}
