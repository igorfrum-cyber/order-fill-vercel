import { FILL_COMPOSITION_ORDER, fillReadiness, MATCH_LAYER_TABS, pairedRowCount } from "../../../features/report/rowPresentation.js";
import { IconCheck } from "../../icons.jsx";
import { Ring } from "../../widgets.jsx";
import { STATUS_META } from "./ReportRow.jsx";

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

export function ReviewSummary({ counts, summary, activeTab, onTab }) {
  const filledCount = counts.filled ?? 0;
  const emptyCount = counts.empty ?? 0;
  const paired = pairedRowCount(counts);
  const unmatchedTotal = MATCH_LAYER_TABS.reduce((sum, key) => sum + (counts[key] || 0), 0);

  return (
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
                  onClick={() => onTab(key)}
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
                <button key={key} type="button" onClick={() => onTab(key)} className="group flex items-center gap-1.5 text-[13px]">
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
                  onClick={() => onTab(card.key)}
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
  );
}
