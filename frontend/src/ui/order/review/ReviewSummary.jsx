import { MATCH_LAYER_TABS, REPORT_TABS } from "../../../features/report/rowPresentation.js";

const TONE = {
  needs_decision: "text-[var(--color-danger)]",
  not_in_source: "text-[var(--color-ink)]",
  check_name_or_volume: "text-[var(--color-ink)]",
  not_in_blank: "text-[var(--color-ink)]",
  to_order: "text-[var(--color-ok)]",
  order_not_needed: "text-[var(--color-ink-soft)]",
};

const SUMMARY_KEYS = ["needs_decision", "to_order", "order_not_needed", "not_in_source", "not_in_blank", "check_name_or_volume"];

export function ReviewSummary({ counts, summary, activeTab, onTab }) {
  const labels = Object.fromEntries(REPORT_TABS.map((tab) => [tab.key, tab.label]));
  const unmatchedTotal = MATCH_LAYER_TABS.reduce((sum, key) => sum + (counts[key] || 0), 0);

  return (
    <div data-tour="fill-summary" className="border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-5">
      <div className="flex flex-col gap-5 xl:flex-row xl:items-start">
        <div className="min-w-0 flex-1">
          <div className="grid gap-x-8 gap-y-2 sm:grid-cols-2 lg:grid-cols-3">
            {SUMMARY_KEYS.map((key) => {
              const n = counts[key] ?? 0;
              const active = activeTab === key;
              return (
                <button
                  key={key}
                  type="button"
                  onClick={() => onTab(key)}
                  className={`flex items-baseline justify-between gap-3 rounded-lg px-1 py-0.5 text-left ${active ? "bg-[var(--color-brand-soft)]" : "hover:bg-[var(--color-line-soft)]"}`}
                >
                  <span className="text-[14px] text-[var(--color-ink-soft)]">{labels[key]}</span>
                  <span className={`font-mono text-[20px] font-semibold tabular-nums ${TONE[key]}`}>{n}</span>
                </button>
              );
            })}
          </div>
          {summary.orderMonthLabel && (
            <div className="mt-3 font-mono text-[13px] text-[var(--color-ink-faint)]">
              {summary.brand}. Заказ на {summary.orderMonthLabel}. Период: {summary.actualMainPeriod || "—"}.
              {summary.cityRule ? ` ${summary.cityRule}: срок поставки ${summary.deliveryWeeks} нед.` : ""}
              {summary.blankDuplicateArticles ? ` Дублей артикулов в бланке: ${summary.blankDuplicateArticles}.` : ""}
            </div>
          )}
        </div>
        <div className="text-[13px] text-[var(--color-ink-faint)] xl:w-48 xl:text-right">
          {unmatchedTotal ? `${unmatchedTotal} без пары в 1С или бланке` : "Все строки с парой"}
        </div>
      </div>
    </div>
  );
}
