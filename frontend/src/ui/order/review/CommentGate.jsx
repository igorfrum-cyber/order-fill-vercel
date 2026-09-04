import { commentGateRows, rowKey } from "../../../features/order/reviewEdits.js";
import {
  commentGateCommentPlaceholder,
  commentGateConfirm,
  commentGateHint,
  commentGateTitle,
  displayArticle,
  displayName,
  quantityDisplay,
} from "../../../features/report/rowPresentation.js";
import { GhostButton, PrimaryButton } from "../../widgets.jsx";

export function CommentGate({ rows, edits, onEdit, onCancel, onConfirm }) {
  const blocked = commentGateRows(rows, edits);
  const canConfirm = blocked.length === 0;

  return (
    <div className="help-modal-backdrop fixed inset-0 z-20 grid place-items-center bg-slate-900/45 p-5" role="dialog" aria-modal="true">
      <div className="help-modal-card w-full max-w-2xl rounded-modal border border-[var(--color-line)] bg-[var(--color-surface)] p-6 shadow-xl">
        <h2 className="text-[18px] font-semibold tracking-tight">{commentGateTitle}</h2>
        <p className="mt-2 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{commentGateHint}</p>
        <div className="mt-4 max-h-[50vh] space-y-3 overflow-auto">
          {rows.map((row, index) => {
            const key = rowKey(row);
            const edit = edits.get(key) || { value: row.inserted ?? "", comment: "" };
            const needsComment = blocked.some((item) => rowKey(item) === key);
            return (
              <label key={key} className="block rounded-lg border border-[var(--color-line)] px-4 py-3">
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                  <div className="min-w-0">
                    <div className="font-mono text-[13px] text-[var(--color-ink-soft)]">{displayArticle(row) || "—"}</div>
                    <div className="text-[15px] font-medium">{displayName(row) || "—"}</div>
                  </div>
                  <div className="font-mono text-[14px] tabular-nums text-[var(--color-ink)]">
                    {quantityDisplay(edit.value) || "0"}
                  </div>
                </div>
                <input
                  type="text"
                  value={edit.comment || ""}
                  autoFocus={index === 0}
                  onChange={(event) => onEdit(key, { comment: event.target.value })}
                  placeholder={commentGateCommentPlaceholder}
                  className={`mt-3 w-full rounded-lg border px-3 py-2 text-[14px] outline-none transition focus:border-[var(--color-brand)] focus:ring-4 focus:ring-[var(--color-brand-soft)] ${
                    needsComment
                      ? "border-[var(--color-warn)] bg-[var(--color-warn-soft)]"
                      : "border-[var(--color-line)] bg-[var(--color-surface)]"
                  }`}
                />
              </label>
            );
          })}
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <GhostButton onClick={onCancel}>Назад</GhostButton>
          <PrimaryButton onClick={onConfirm} disabled={!canConfirm}>
            {commentGateConfirm}
          </PrimaryButton>
        </div>
      </div>
    </div>
  );
}
