import { useMemo, useState } from "react";
import { adjustmentLabelForBrand } from "../../features/brands/brandPresentation.js";
import { rowKey } from "../../features/order/reviewEdits.js";
import {
  canProceedPastDuplicates,
  countByTab,
  FILL_COMPOSITION_ORDER,
  fillReadiness,
  MATCH_LAYER_TABS,
  matchLayerHint,
  pairedRowCount,
  presentationStatus,
  visibleFillTabs,
  visibleReportRows,
} from "../../features/report/rowPresentation.js";
import { IconCheck, IconDownload } from "../icons.jsx";
import { GhostButton, PrimaryButton, Ring } from "../widgets.jsx";
import { STATUS_META } from "./review/ReportRow.jsx";
import { ReviewTable } from "./review/ReviewTable.jsx";
import { ReviewTabs } from "./review/ReviewTabs.jsx";

const COMPOSITION = {
  filled: "bg-[var(--color-ok)]",
  empty: "bg-[var(--color-warn)]",
  check: "bg-[color-mix(in_srgb,var(--color-warn)_55%,white)]",
  duplicate: "bg-[var(--color-danger)]",
};

const MATCH_CARDS = [
  { key: "not_in_table", label: "Нет в таблице", detail: "бланк без пары в заказе", bar: "bg-[var(--color-neutral)]" },
  { key: "not_in_blank", label: "Нет в бланке", detail: "заказ без пары в бланке", bar: "bg-[color-mix(in_srgb,var(--color-neutral)_55%,white)]" },
];

export function FillStage({
  brand,
  rows,
  edits,
  onEdit,
  invalidKeys,
  summary,
  status,
  busy,
  onDownloadFiles,
  onIssueReport,
}) {
  const [tab, setTab] = useState("empty");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState(null);
  const [acknowledgedDuplicates, setAcknowledgedDuplicates] = useState(() => new Set());
  const counts = useMemo(() => countByTab(rows), [rows]);
  const filledCount = counts.filled ?? 0;
  const emptyCount = counts.empty ?? 0;
  const duplicateCount = counts.duplicate ?? 0;
  const duplicateKeys = useMemo(
    () => rows.filter((row) => presentationStatus(row) === "duplicate").map(rowKey),
    [rows],
  );
  const paired = pairedRowCount(counts);
  const tabs = useMemo(() => visibleFillTabs(counts), [counts]);
  const activeTab = tabs.some((item) => item.key === tab) ? tab : (tabs[0]?.key ?? "all");
  const visible = useMemo(() => visibleReportRows(rows, { tab: activeTab, query }), [rows, activeTab, query]);
  const boxLabel = summary.adjustmentLabel || adjustmentLabelForBrand(brand);
  const canProceed = canProceedPastDuplicates({ duplicateKeys, acknowledgedKeys: acknowledgedDuplicates });
  const acknowledgedCount = duplicateKeys.filter((key) => acknowledgedDuplicates.has(key)).length;
  const unmatchedTotal = MATCH_LAYER_TABS.reduce((sum, key) => sum + (counts[key] || 0), 0);
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
      <div className="border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-5">
        <div className="flex flex-col gap-6 xl:flex-row xl:items-start">
          <div className="flex shrink-0 items-center gap-4">
            <Ring value={fillReadiness(counts)} />
            <div>
              <div className="text-[14px] font-medium text-[var(--color-ink-soft)]">Готовность бланка</div>
              <div className="mt-0.5 flex items-baseline gap-1.5">
                <span className="font-mono text-[28px] font-semibold leading-none tabular-nums">{filledCount}</span>
                <span className="font-mono text-[15px] text-[var(--color-ink-faint)]">/ {paired}</span>
              </div>
              <div className="mt-1 text-[13px] text-[var(--color-ink-faint)]">с парой заполнено</div>
              {emptyCount > 0 && (
                <div className="mt-0.5 text-[13px] text-[var(--color-ink-faint)]">{emptyCount} ещё пустые</div>
              )}
            </div>
          </div>

          <div className="hidden w-px self-stretch bg-[var(--color-line)] xl:block" />

          <div className="min-w-0 flex-1">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-[14px] font-medium text-[var(--color-ink-soft)]">Заполнение</span>
              <span className="font-mono text-[13px] text-[var(--color-ink-faint)]">{paired} с парой</span>
            </div>
            <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-[var(--color-line-soft)]">
              {FILL_COMPOSITION_ORDER.map((key) => {
                const n = counts[key] ?? 0;
                if (!n || !paired) return null;
                return (
                  <button
                    key={key}
                    type="button"
                    title={`${STATUS_META[key].label}: ${n}`}
                    onClick={() => setTab(key)}
                    style={{ width: `${(n / paired) * 100}%` }}
                    className={`h-full ${COMPOSITION[key]} transition-opacity hover:opacity-80`}
                  />
                );
              })}
            </div>
            <div className="mt-2.5 flex flex-wrap gap-x-4 gap-y-1.5">
              {FILL_COMPOSITION_ORDER.map((key) => {
                const n = counts[key] ?? 0;
                if (!n) return null;
                return (
                  <button key={key} type="button" onClick={() => setTab(key)} className="group flex items-center gap-1.5 text-[13px]">
                    <span className={`h-2 w-2 rounded-[3px] ${COMPOSITION[key]}`} />
                    <span className="text-[var(--color-ink-soft)] group-hover:text-[var(--color-ink)]">{STATUS_META[key].label}</span>
                    <span className="font-mono tabular-nums text-[var(--color-ink-faint)]">{n}</span>
                  </button>
                );
              })}
            </div>
            {summary.orderMonthLabel && (
              <div className="mt-2 font-mono text-[13px] text-[var(--color-ink-faint)]">
                {summary.brand}. Заказ на {summary.orderMonthLabel}. Период: {summary.actualMainPeriod || "—"}.
                {summary.cityRule ? ` ${summary.cityRule}: срок поставки ${summary.deliveryWeeks} нед.` : ""}
                {summary.blankDuplicateArticles ? ` Дублей артикулов в бланке: ${summary.blankDuplicateArticles}.` : ""}
              </div>
            )}
          </div>

          <div className="hidden w-px self-stretch bg-[var(--color-line)] xl:block" />

          <div className="w-full shrink-0 xl:w-96">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-[14px] font-medium text-[var(--color-ink-soft)]">Сопоставление</span>
              {unmatchedTotal === 0 ? (
                <span className="flex items-center gap-1 text-[13px] text-[var(--color-ok)]">
                  <IconCheck className="h-3.5 w-3.5" />
                  все нашли пару
                </span>
              ) : (
                <span className="font-mono text-[13px] text-[var(--color-ink-faint)]">{unmatchedTotal} без пары</span>
              )}
            </div>
            <div className="grid grid-cols-2 gap-2">
              {MATCH_CARDS.map((card) => {
                const n = counts[card.key] ?? 0;
                const active = activeTab === card.key;
                return (
                  <button
                    key={card.key}
                    type="button"
                    onClick={() => setTab(card.key)}
                    className={`flex gap-2.5 rounded-xl border px-3 py-2.5 text-left transition ${
                      active
                        ? "border-[var(--color-brand)] bg-[var(--color-brand-soft)]"
                        : n
                          ? "border-[color-mix(in_srgb,var(--color-neutral)_35%,white)] bg-[var(--color-neutral-soft)] hover:border-[var(--color-neutral)]"
                          : "border-[var(--color-line)] bg-[var(--color-surface)] hover:bg-[var(--color-line-soft)]"
                    }`}
                  >
                    <span className={`mt-0.5 h-9 w-1 shrink-0 rounded-full ${card.bar}`} />
                    <span className="min-w-0">
                      <span className="block text-[13px] text-[var(--color-ink-soft)]">{card.label}</span>
                      <span className="mt-0.5 block font-mono text-[20px] font-semibold leading-none tabular-nums">{n}</span>
                      <span className="mt-1.5 block text-[12px] leading-snug text-[var(--color-ink-faint)]">{card.detail}</span>
                    </span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>

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
          <PrimaryButton onClick={onDownloadFiles} disabled={busy || !canProceed}>
            <span className={`h-2 w-2 rounded-full ${canProceed ? "bg-[var(--color-ok)]" : "bg-white/40"}`} />
            {busy ? "Готовлю файлы..." : "Проверить файлы"}
            <IconDownload className="h-4 w-4" />
          </PrimaryButton>
        </div>
      </footer>
    </div>
  );
}
