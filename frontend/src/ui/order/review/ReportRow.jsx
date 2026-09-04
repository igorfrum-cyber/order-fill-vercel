import { editRequiresComment, normalizeOrderValue } from "../../../features/order/editRules.js";
import { quantityDivergesFromRecommendation, roundingComment } from "../../../features/order/quantityPresentation.js";
import { rowKey } from "../../../features/order/reviewEdits.js";
import { duplicateDescription } from "../../../features/report/issueReport.js";
import { baselineForReportRow, statusLabel } from "../../../features/report/reportModel.js";
import { boxStep, displayArticle, displayName, matchPercent, presentationStatus, quantityDisplay, attentionReason } from "../../../features/report/rowPresentation.js";
import { IconChevron } from "../../icons.jsx";
import { Stepper } from "../../widgets.jsx";

export const STATUS_META = {
  filled: { label: "Заполнено", tone: "ok", bar: "bg-[var(--color-ok)]" },
  empty: { label: "Пусто", tone: "warn", bar: "bg-[var(--color-warn)]" },
  check: { label: "Нужно проверить", tone: "warn", bar: "bg-[var(--color-warn)]" },
  duplicate: { label: "Дубли", tone: "danger", bar: "bg-[var(--color-danger)]" },
  not_in_table: { label: "Нет в таблице", tone: "neutral", bar: "bg-[var(--color-neutral)]" },
  not_in_blank: { label: "Нет в бланке", tone: "neutral", bar: "bg-[var(--color-neutral)]" },
};

const TONE_CHIP = {
  ok: "text-[var(--color-ok)] bg-[var(--color-ok-soft)]",
  warn: "text-[var(--color-warn)] bg-[var(--color-warn-soft)]",
  danger: "text-[var(--color-danger)] bg-[var(--color-danger-soft)]",
  neutral: "text-[var(--color-neutral)] bg-[var(--color-neutral-soft)]",
};

export function ReportRow({ row, edit, expanded, invalid, acknowledged, boxLabel, onToggle, onEdit, onAcknowledge }) {
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
  const reason = attentionReason(row);

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
            <span className="relative inline-flex">
              {String(edit?.comment || "").trim() ? (
                <span
                  title={edit.comment}
                  aria-label="Есть комментарий"
                  className="pointer-events-none absolute -right-0.5 -top-0.5 z-10 h-0 w-0 border-l-8 border-t-8 border-l-transparent border-t-[#ea580c]"
                />
              ) : null}
              <Stepper
                value={edit?.value ?? ""}
                disabled={row.editable === false}
                onChange={(value) => onEdit(key, { value })}
                step={boxStep(row)}
              />
            </span>
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
            {reason ? (
              <p className="mb-3 pl-5 text-[14px] leading-relaxed text-[var(--color-ink)]">{reason}</p>
            ) : null}
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
              {row.sourceComment ? (
                <Detail label="Комментарий в таблице">{row.sourceComment}</Detail>
              ) : null}
              {edit?.comment && edit.comment !== row.sourceComment ? (
                <Detail label="Комментарий">{edit.comment}</Detail>
              ) : null}
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

export function rowNeedsComment(row, edit) {
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
