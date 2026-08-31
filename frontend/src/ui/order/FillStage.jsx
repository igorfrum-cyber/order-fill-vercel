import { useMemo, useState } from "react";
import { adjustmentLabelForBrand } from "../../features/brands/brandPresentation.js";
import { editRequiresComment, normalizeOrderValue } from "../../features/order/editRules.js";
import { quantityDivergesFromRecommendation, roundingComment } from "../../features/order/quantityPresentation.js";
import { rowKey } from "../../features/order/reviewEdits.js";
import { duplicateDescription } from "../../features/report/issueReport.js";
import { baselineForReportRow, statusLabel } from "../../features/report/reportModel.js";
import {
  boxStep,
  canProceedPastDuplicates,
  countByTab,
  displayArticle,
  displayName,
  FILL_COMPOSITION_ORDER,
  fillReadiness,
  MATCH_LAYER_TABS,
  matchLayerHint,
  matchPercent,
  pairedRowCount,
  presentationStatus,
  quantityDisplay,
  visibleFillTabs,
  visibleReportRows,
} from "../../features/report/rowPresentation.js";
import { IconCheck, IconChevron, IconDownload, IconSearch, IconX } from "../icons.jsx";
import { GhostButton, PrimaryButton, Ring, Stepper } from "../widgets.jsx";

const STATUS_META = {
  filled: { label: "Заполнено", tone: "ok", bar: "bg-[var(--color-ok)]" },
  empty: { label: "Пусто", tone: "warn", bar: "bg-[var(--color-warn)]" },
  check: { label: "Нужно проверить", tone: "warn", bar: "bg-[var(--color-warn)]" },
  duplicate: { label: "Дубли", tone: "danger", bar: "bg-[var(--color-danger)]" },
  not_in_table: { label: "Нет в таблице", tone: "neutral", bar: "bg-[var(--color-neutral)]" },
  not_in_blank: { label: "Нет в бланке", tone: "neutral", bar: "bg-[var(--color-neutral)]" },
};

const COMPOSITION = {
  filled: "bg-[var(--color-ok)]",
  empty: "bg-[var(--color-warn)]",
  check: "bg-[color-mix(in_srgb,var(--color-warn)_55%,white)]",
  duplicate: "bg-[var(--color-danger)]",
};

const TONE_CHIP = {
  ok: "text-[var(--color-ok)] bg-[var(--color-ok-soft)]",
  warn: "text-[var(--color-warn)] bg-[var(--color-warn-soft)]",
  danger: "text-[var(--color-danger)] bg-[var(--color-danger-soft)]",
  neutral: "text-[var(--color-neutral)] bg-[var(--color-neutral-soft)]",
};

const MATCH_CARDS = [
  { key: "not_in_table", label: "Нет в таблице", detail: "бланк без пары в заказе", bar: "bg-[var(--color-neutral)]" },
  { key: "not_in_blank", label: "Нет в бланке", detail: "заказ без пары в бланке", bar: "bg-[color-mix(in_srgb,var(--color-neutral)_55%,white)]" },
];

const TABLE_HEADERS = [
  { key: "bar", label: "", align: "left" },
  { key: "article", label: "Артикул", align: "left" },
  { key: "name", label: "Товар", align: "left" },
  { key: "unit", label: "Объём", align: "right" },
  { key: "stock", label: "Остаток", align: "right" },
  { key: "transit", label: "В пути", align: "right" },
  { key: "recommended", label: "Реком.", align: "right" },
  { key: "inserted", label: "Вставлено", align: "right" },
  { key: "match", label: "Совпад.", align: "right" },
  { key: "comment", label: "Комментарий", align: "left" },
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

      <div className="flex flex-wrap items-center gap-3 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-2.5">
        <div className="flex flex-wrap gap-1">
          {tabs.map((item) => {
            const n = counts[item.key] ?? 0;
            const active = activeTab === item.key;
            return (
              <button
                key={item.key}
                type="button"
                onClick={() => setTab(item.key)}
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
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Артикул или наименование"
            className="w-full rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] py-2 pl-8 pr-8 text-[14px] outline-none transition focus:border-[var(--color-brand)] focus:ring-4 focus:ring-[var(--color-brand-soft)]"
          />
          {query && (
            <button type="button" onClick={() => setQuery("")} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--color-ink-faint)] hover:text-[var(--color-ink)]">
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

      <div className="flex-1 overflow-auto px-6 py-4">
        <table className="w-full min-w-[1280px] table-fixed border-separate border-spacing-0">
          <colgroup>
            <col className="w-12" />
            <col className="w-[11%]" />
            <col />
            <col className="w-[8%]" />
            <col className="w-[8%]" />
            <col className="w-[8%]" />
            <col className="w-[9%]" />
            <col className="w-[13%]" />
            <col className="w-[8%]" />
            <col className="w-[18%]" />
          </colgroup>
          <thead>
            <tr className="text-left">
              {TABLE_HEADERS.map((header) => (
                <th
                  key={header.key}
                  className={`sticky top-0 z-10 whitespace-nowrap bg-[var(--color-ground)] px-4 pb-3.5 pt-2 text-[13px] font-medium tracking-wide text-[var(--color-ink-faint)] ${
                    header.align === "right" ? "text-right" : ""
                  }`}
                >
                  {header.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {visible.length === 0 && (
              <tr>
                <td colSpan={10} className="py-16 text-center text-[15px] text-[var(--color-ink-faint)]">
                  Нет позиций в этой категории
                </td>
              </tr>
            )}
            {visible.map((row) => (
              <ReportRow
                key={rowKey(row)}
                row={row}
                edit={edits.get(rowKey(row))}
                expanded={expanded === rowKey(row)}
                invalid={invalidKeys.has(rowKey(row))}
                acknowledged={acknowledgedDuplicates.has(rowKey(row))}
                boxLabel={boxLabel}
                onToggle={() => setExpanded(expanded === rowKey(row) ? null : rowKey(row))}
                onEdit={onEdit}
                onAcknowledge={(next) => toggleDuplicateAck(rowKey(row), next)}
              />
            ))}
          </tbody>
        </table>
      </div>

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

function ReportRow({ row, edit, expanded, invalid, acknowledged, boxLabel, onToggle, onEdit, onAcknowledge }) {
  const status = presentationStatus(row);
  const meta = STATUS_META[status];
  const diverges = row.editable !== false && quantityDivergesFromRecommendation(row, edit?.value);
  const note = roundingComment(row);
  const cell = `border-b border-[var(--color-line-soft)] px-4 py-4 align-middle text-[14px] ${invalid ? "bg-[var(--color-danger-soft)]" : acknowledged ? "bg-[var(--color-ok-soft)]" : diverges ? "bg-[var(--color-warn-soft)]" : ""}`;
  const num = `${cell} text-right font-mono tabular-nums`;
  const key = rowKey(row);
  const match = matchPercent(row);
  const needsComment = rowNeedsComment(row, edit);
  const isDuplicate = status === "duplicate";

  return (
    <>
      <tr className="group transition hover:bg-[var(--color-line-soft)]">
        <td className={`${cell} pl-1 pr-2`}>
          <button type="button" onClick={onToggle} className="flex items-center gap-2">
            <span className={`h-7 w-1.5 rounded-full ${meta.bar}`} aria-label={meta.label} />
            <IconChevron className={`h-4 w-4 text-[var(--color-ink-faint)] transition ${expanded ? "rotate-180" : ""}`} />
          </button>
        </td>
        <td className={`${cell} truncate font-mono text-[13px] text-[var(--color-ink-soft)]`} title={displayArticle(row)}>
          {displayArticle(row) || "—"}
        </td>
        <td className={cell}>
          <span className="block leading-snug font-medium">{displayName(row) || "—"}</span>
          {isDuplicate && (
            <span className={`mt-1 inline-flex rounded-md px-1.5 py-0.5 text-[12px] font-medium ${acknowledged ? TONE_CHIP.ok : TONE_CHIP.danger}`}>
              {acknowledged ? "дубль подтверждён" : "дубль"}
            </span>
          )}
        </td>
        <td className={`${num} text-[var(--color-ink-soft)]`}>{row.blankUnit || "—"}</td>
        <td className={num}>{quantityDisplay(row.stock) || "—"}</td>
        <td className={`${num} text-[var(--color-ink-soft)]`}>{quantityDisplay(row.inTransit) || "—"}</td>
        <td className={`${num} font-semibold text-[var(--color-brand-strong)]`}>
          {row.recommended == null ? "—" : Number(row.recommended).toFixed(2)}
        </td>
        <td className={cell}>
          <div className="flex items-center justify-end gap-2.5">
            {note ? (
              <span className="hidden max-w-28 truncate rounded-md bg-[var(--color-warn-soft)] px-1.5 py-0.5 text-[12px] font-medium text-[var(--color-warn)] xl:inline" title={note}>
                {note}
              </span>
            ) : null}
            <Stepper
              value={edit?.value ?? ""}
              disabled={row.editable === false}
              onChange={(value) => onEdit(key, { value })}
              step={boxStep(row)}
            />
          </div>
        </td>
        <td className={cell}>
          <div className="flex justify-end">
            {match == null ? <span className="px-2 text-[var(--color-ink-faint)]">—</span> : <MatchPill value={match} />}
          </div>
        </td>
        <td className={cell}>
          {row.editable === false ? (
            <span className="block max-w-xs truncate text-[13px] text-[var(--color-ink-soft)]" title={edit?.comment || ""}>
              {edit?.comment || "—"}
            </span>
          ) : (
            <input
              type="text"
              value={edit?.comment || ""}
              onChange={(event) => onEdit(key, { comment: event.target.value })}
              placeholder="Почему изменили количество"
              className={`w-full rounded-lg border px-3 py-2 text-[13px] outline-none ${
                needsComment ? "border-[var(--color-warn)] bg-[var(--color-warn-soft)]" : "border-[var(--color-line)] bg-[var(--color-surface)]"
              }`}
            />
          )}
        </td>
      </tr>
      {isDuplicate && (
        <tr>
          <td colSpan={10} className={`border-b border-[var(--color-line-soft)] px-4 py-3 ${acknowledged ? "bg-[var(--color-ok-soft)]" : "bg-[var(--color-danger-soft)]"}`}>
            <div className="flex flex-wrap items-center justify-between gap-3 pl-8">
              <div className="min-w-0 text-[14px]">
                <div className="font-medium text-[var(--color-ink)]">Конфликт в таблице заказа</div>
                <div className="mt-0.5 text-[13px] text-[var(--color-ink-soft)]">
                  {duplicateDescription(row.duplicateCandidates) || "несколько строк с одним артикулом"}
                </div>
              </div>
              <label className="flex cursor-pointer items-center gap-2 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] px-3 py-2 text-[14px] font-medium text-[var(--color-ink)]">
                <input
                  type="checkbox"
                  checked={acknowledged}
                  onChange={(event) => onAcknowledge(event.target.checked)}
                  className="h-4 w-4 rounded border-[var(--color-line)] accent-[var(--color-brand)]"
                />
                Оставляю как есть
              </label>
            </div>
          </td>
        </tr>
      )}
      {expanded && (
        <tr>
          <td colSpan={10} className="border-b border-[var(--color-line-soft)] bg-[var(--color-ground)] px-3 py-3">
            <div className="grid gap-x-8 gap-y-3 pl-5 text-[14px] sm:grid-cols-3">
              <Detail label="Статус">
                <span className={`inline-flex rounded-md px-2 py-0.5 text-[13px] font-medium ${TONE_CHIP[meta.tone]}`}>
                  {statusLabel(row.status)}
                </span>
              </Detail>
              <Detail label={boxLabel}>{quantityDisplay(row.blankBoxSize) || "—"}</Detail>
              <Detail label="Заказано по факту">{row.hasOrderedFact ? quantityDisplay(row.orderedFact) : "—"}</Detail>
              <Detail label="Бланк">
                <span className="font-mono text-[13px]">{row.blankLabel || "—"}</span>
              </Detail>
              {row.sourceArticle || row.sourceName ? (
                <Detail label="Таблица заказа">
                  <span className="font-mono text-[13px]">
                    {[row.sourceArticle, row.sourceName].filter(Boolean).join(" · ") || "—"}
                  </span>
                </Detail>
              ) : null}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

function Detail({ label, children }) {
  return (
    <div>
      <div className="mb-0.5 text-[13px] text-[var(--color-ink-faint)]">{label}</div>
      <div>{children}</div>
    </div>
  );
}

function MatchPill({ value }) {
  const tone = value >= 85 ? "ok" : value >= 60 ? "warn" : "danger";
  return (
    <span className={`rounded-md px-2 py-0.5 font-mono text-[13px] font-medium tabular-nums ${TONE_CHIP[tone]}`}>
      {value}%
    </span>
  );
}

function rowNeedsComment(row, edit) {
  if (!edit || row.editable === false) return false;
  let value;
  try {
    value = normalizeOrderValue(edit.value);
  } catch {
    return false;
  }
  const initial = row.inserted == null ? null : Number(row.inserted);
  return editRequiresComment({
    value,
    baseline: baselineForReportRow(row),
    initial,
    comment: edit.comment,
    autoComment: row.autoComment,
  });
}
