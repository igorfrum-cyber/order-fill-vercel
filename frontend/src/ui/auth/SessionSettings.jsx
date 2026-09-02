import { useEffect, useState } from "react";
import { listSessions, logoutEverywhere, revokeSession } from "../../api/auth.js";
import { formatSessionWhen, sessionIsPhone } from "../../features/auth/session.js";
import {
  logoutEverywhereConfirm,
  logoutEverywhereLabel,
  sessionCurrentLabel,
  sessionRevokeLabel,
  sessionsHint,
  sessionsTitle,
} from "../../features/help/copy.js";
import { IconComputer, IconPhone } from "../icons.jsx";
import { GhostButton, Modal } from "../widgets.jsx";

export function SessionSettings({ onSignedOut }) {
  const [items, setItems] = useState([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirmEverywhere, setConfirmEverywhere] = useState(false);

  async function refresh() {
    const result = await listSessions();
    setItems(result.sessions || []);
  }

  useEffect(() => {
    refresh().catch(() => setError("Не удалось загрузить активные входы."));
  }, []);

  async function revoke(item) {
    setBusy(true);
    setError("");
    try {
      await revokeSession(item.id);
      if (item.current) {
        onSignedOut?.();
        return;
      }
      await refresh();
    } catch {
      setError("Не удалось закрыть вход.");
    } finally {
      setBusy(false);
    }
  }

  async function signOutEverywhere() {
    setBusy(true);
    setError("");
    try {
      await logoutEverywhere();
      onSignedOut?.();
    } catch {
      setError("Не удалось выйти со всех устройств.");
      setConfirmEverywhere(false);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
      <div>
        <h2 className="text-[16px] font-semibold">{sessionsTitle}</h2>
        <p className="mt-1 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{sessionsHint}</p>
      </div>
      {items.length ? (
        <ul className="space-y-2">
          {items.map((item) => (
            <li
              key={item.id}
              className={`flex items-center gap-3 rounded-xl px-3 py-3 ${
                item.current ? "bg-[var(--color-brand-soft)]" : "bg-[var(--color-ground)]"
              }`}
            >
              <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-[var(--color-surface)] text-[var(--color-brand)]">
                {sessionIsPhone(item.device) ? <IconPhone className="h-5 w-5" /> : <IconComputer className="h-5 w-5" />}
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-[15px] font-medium">{item.device}</p>
                <p className="text-[13px] text-[var(--color-ink-faint)]">
                  {item.current ? sessionCurrentLabel : `вход ${formatSessionWhen(item.created_at)}`}
                </p>
              </div>
              {item.current ? (
                <span className="shrink-0 rounded-full bg-[var(--color-surface)] px-2.5 py-1 text-[12px] font-semibold text-[var(--color-brand-strong)]">
                  сейчас
                </span>
              ) : (
                <GhostButton type="button" onClick={() => revoke(item)} disabled={busy}>
                  {sessionRevokeLabel}
                </GhostButton>
              )}
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-[14px] text-[var(--color-ink-soft)]">Сейчас виден только этот вход.</p>
      )}
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <GhostButton onClick={() => setConfirmEverywhere(true)} disabled={busy}>
        {logoutEverywhereLabel}
      </GhostButton>
      {confirmEverywhere ? (
        <Modal
          title={logoutEverywhereLabel}
          cancelLabel="Отмена"
          confirmLabel="Выйти"
          confirmDisabled={busy}
          onCancel={() => setConfirmEverywhere(false)}
          onConfirm={signOutEverywhere}
        >
          {logoutEverywhereConfirm}
        </Modal>
      ) : null}
    </div>
  );
}
