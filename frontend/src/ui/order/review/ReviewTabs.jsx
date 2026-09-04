import { IconSearch, IconX } from "../../icons.jsx";

export function ReviewTabs({ tabs, counts, activeTab, query, onTab, onQuery, duplicateCount, acknowledgedCount, hint }) {
  return (
    <>
      <div data-tour="fill-tabs" className="flex flex-wrap items-center gap-3 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-2.5">
        <div className="flex flex-wrap gap-1">
          {tabs.map((item) => {
            const n = counts[item.key] ?? 0;
            const active = activeTab === item.key;
            return (
              <button
                key={item.key}
                type="button"
                onClick={() => onTab(item.key)}
                className={`flex items-center gap-1.5 rounded-lg px-3 py-2 text-[14px] font-medium transition ${
                  active ? "bg-[var(--color-brand)] text-white" : "text-[var(--color-ink-soft)] hover:bg-[var(--color-line-soft)]"
                }`}
              >
                {item.label}
                <span className={`rounded-full px-1.5 font-mono text-[12px] tabular-nums ${active ? "bg-white/20" : "bg-[var(--color-neutral-soft)] text-[var(--color-ink-faint)]"}`}>
                  {n}
                </span>
              </button>
            );
          })}
        </div>
        <div className="relative ml-auto min-w-52">
          <IconSearch className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--color-ink-faint)]" />
          <input
            value={query}
            onChange={(event) => onQuery(event.target.value)}
            placeholder="Артикул или наименование"
            className="w-full rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] py-2 pl-8 pr-8 text-[14px] outline-none transition focus:border-[var(--color-brand)] focus:ring-4 focus:ring-[var(--color-brand-soft)]"
          />
          {query && (
            <button type="button" onClick={() => onQuery("")} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--color-ink-faint)] hover:text-[var(--color-ink)]">
              <IconX className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {activeTab === "duplicate" && duplicateCount > 0 ? (
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[color-mix(in_srgb,var(--color-danger)_25%,white)] bg-[var(--color-danger-soft)] px-6 py-2.5 text-[14px] text-[var(--color-ink)]">
          <span>В таблице заказа несколько строк на одну позицию бланка. На каждой строке отметьте «оставляю», когда разобрали конфликт.</span>
          <span className="font-mono text-[13px] tabular-nums text-[var(--color-ink-soft)]">
            {acknowledgedCount} из {duplicateCount} подтверждены
          </span>
        </div>
      ) : hint ? (
        <div className="border-b border-[var(--color-line)] bg-[var(--color-neutral-soft)] px-6 py-2.5 text-[14px] text-[var(--color-ink-soft)]">
          {hint}
        </div>
      ) : null}
    </>
  );
}
