import { useMemo, useState } from "react";
import { adjustmentLabelForBrand } from "../../features/brands/brandPresentation.js";
import { rowKey } from "../../features/order/reviewEdits.js";
import {
  canProceedPastDuplicates,
  countByTab,
  matchLayerHint,
  presentationStatus,
  visibleFillTabs,
  visibleReportRows,
} from "../../features/report/rowPresentation.js";
import { IconDownload } from "../icons.jsx";
import { GhostButton, PrimaryButton } from "../widgets.jsx";
import { ReviewSummary } from "./review/ReviewSummary.jsx";
import { ReviewTable } from "./review/ReviewTable.jsx";
import { ReviewTabs } from "./review/ReviewTabs.jsx";

export function FillStage({
  brand,
  rows,
  edits,
  onEdit,
  invalidKeys,
  summary,
  status,
  busy,
  banner,
  onDownloadFiles,
  onIssueReport,
}) {
  const [tab, setTab] = useState("empty");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState(null);
  const [acknowledgedDuplicates, setAcknowledgedDuplicates] = useState(() => new Set());
  const counts = useMemo(() => countByTab(rows), [rows]);
  const duplicateCount = counts.duplicate ?? 0;
  const duplicateKeys = useMemo(
    () => rows.filter((row) => presentationStatus(row) === "duplicate").map(rowKey),
    [rows],
  );
  const tabs = useMemo(() => visibleFillTabs(counts), [counts]);
  const activeTab = tabs.some((item) => item.key === tab) ? tab : (tabs[0]?.key ?? "all");
  const visible = useMemo(() => visibleReportRows(rows, { tab: activeTab, query }), [rows, activeTab, query]);
  const boxLabel = summary.adjustmentLabel || adjustmentLabelForBrand(brand);
  const canProceed = canProceedPastDuplicates({ duplicateKeys, acknowledgedKeys: acknowledgedDuplicates });
  const acknowledgedCount = duplicateKeys.filter((key) => acknowledgedDuplicates.has(key)).length;
  const hint = matchLayerHint(activeTab);

  function toggleDuplicateAck(key, next) {
    setAcknowledgedDuplicates((prev) => {
      const copy = new Set(prev);
      if (next) copy.add(key);
      else copy.delete(key);
      return copy;
    });
  }

  return (
    <div className="flex h-full flex-col">
      <ReviewSummary counts={counts} summary={summary} activeTab={activeTab} onTab={setTab} />

      <ReviewTabs
        tabs={tabs}
        counts={counts}
        activeTab={activeTab}
        query={query}
        onTab={setTab}
        onQuery={setQuery}
        duplicateCount={duplicateCount}
        acknowledgedCount={acknowledgedCount}
        hint={hint}
      />

      <ReviewTable
        rows={visible}
        edits={edits}
        expanded={expanded}
        invalidKeys={invalidKeys}
        acknowledgedDuplicates={acknowledgedDuplicates}
        boxLabel={boxLabel}
        onToggle={(key) => setExpanded(expanded === key ? null : key)}
        onEdit={onEdit}
        onAcknowledge={toggleDuplicateAck}
      />

      {banner ? (
        <div
          role="alert"
          className="mx-6 mt-3 rounded-lg border border-[var(--color-danger)]/25 bg-[var(--color-danger-soft)] px-4 py-3 text-[14px] text-[var(--color-danger)]"
        >
          {banner}
        </div>
      ) : null}

      <footer className="flex flex-wrap items-center gap-3 border-t border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-3">
        <GhostButton onClick={onIssueReport} disabled={busy}>
          <IconDownload className="h-4 w-4" />
          Отчёт для 1С
        </GhostButton>
        <div className="ml-auto flex flex-wrap items-center gap-3">
          <span className="font-mono text-[13px] text-[var(--color-ink-soft)]">
            {status ? (
              <span>{status}</span>
            ) : canProceed ? (
              <span className="text-[var(--color-ok)]">{duplicateCount ? "Дубли подтверждены" : "Критичных проблем нет"}</span>
            ) : (
              <button type="button" className="text-[var(--color-danger)] hover:underline" onClick={() => setTab("duplicate")}>
                Сначала подтвердите дубли: {duplicateCount - acknowledgedCount}
              </button>
            )}
          </span>
          <PrimaryButton dataTour="fill-next" onClick={onDownloadFiles} disabled={busy || !canProceed}>
            <span className={`h-2 w-2 rounded-full ${canProceed ? "bg-[var(--color-ok)]" : "bg-white/40"}`} />
            {busy ? "Готовлю файлы..." : "Проверить файлы"}
            <IconDownload className="h-4 w-4" />
          </PrimaryButton>
        </div>
      </footer>
    </div>
  );
}
